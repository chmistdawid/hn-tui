package ui

import (
	"fmt"
	"log"
	"sync"

	"github.com/chmistdawid/hn-tui/internal/api"
	"github.com/chmistdawid/hn-tui/internal/models"
	"github.com/chmistdawid/hn-tui/internal/utils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type appState struct {
	app            *tview.Application
	posts          []models.Post
	feed           string
	offset         int
	pageSize       int
	totalPosts     int
	loadingMore    bool
	selectedPostID int
	cache          map[int][]models.Comment
	cacheMu        sync.RWMutex
	list           *tview.List
	detailView     *tview.TextView
	statusBar      *tview.TextView
}

func SetupMainUI(app *tview.Application, posts []models.Post, total int) {
	state := &appState{
		app:        app,
		posts:      posts,
		feed:       api.FeedTop,
		offset:     len(posts),
		pageSize:   30,
		totalPosts: total,
		cache:      make(map[int][]models.Comment),
	}

	state.list = tview.NewList()
	state.list.SetBorder(true).
		SetTitle(" 🔥 Hacker News ").
		SetTitleColor(tcell.ColorOrange).
		SetBorderColor(tcell.ColorOrange).
		SetBorderPadding(0, 0, 1, 1)
	state.list.SetMainTextColor(tcell.ColorWhite)
	state.list.SetSecondaryTextColor(tcell.ColorGray)
	state.list.SetSelectedTextColor(tcell.ColorWhite)
	state.list.SetSelectedBackgroundColor(tcell.ColorDarkCyan)
	state.list.SetHighlightFullLine(true)
	state.list.ShowSecondaryText(true)
	state.list.SetWrapAround(false)

	state.detailView = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true)
	state.detailView.SetBorder(true).
		SetTitle(" 📝 Details ").
		SetTitleColor(tcell.ColorDodgerBlue).
		SetBorderColor(tcell.ColorDodgerBlue).
		SetBorderPadding(0, 0, 1, 1)

	state.statusBar = tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetMaxLines(1)

	state.populateList()
	state.setupInputCapture()
	state.list.SetChangedFunc(state.onSelectionChanged)

	if len(posts) > 0 {
		state.list.SetCurrentItem(0)
	}

	state.updateStatusBar()

	mainContent := tview.NewFlex().
		AddItem(state.list, 0, 2, true).
		AddItem(state.detailView, 0, 1, false)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(mainContent, 0, 1, true).
		AddItem(state.statusBar, 1, 1, false)

	app.SetRoot(flex, true).SetFocus(state.list)
}

func (s *appState) populateList() {
	s.list.Clear()
	for i, post := range s.posts {
		title := fmt.Sprintf("[white::b]%d. [yellow]▲ %d [white]%s", i+1, post.Score, post.Title)
		ago := utils.FormatTimeAgo(post.Time)
		secondary := fmt.Sprintf("[gray]by [::i]%s[gray:-] | [dodgerblue]%d comments[gray] | %s", post.Author, post.Comments, ago)
		s.list.AddItem(title, secondary, 0, nil)
	}
}

func (s *appState) appendPosts(posts []models.Post) {
	start := len(s.posts)
	s.posts = append(s.posts, posts...)
	for i, post := range posts {
		idx := start + i
		title := fmt.Sprintf("[white::b]%d. [yellow]▲ %d [white]%s", idx+1, post.Score, post.Title)
		ago := utils.FormatTimeAgo(post.Time)
		secondary := fmt.Sprintf("[gray]by [::i]%s[gray:-] | [dodgerblue]%d comments[gray] | %s", post.Author, post.Comments, ago)
		s.list.AddItem(title, secondary, 0, nil)
	}
}

func (s *appState) onSelectionChanged(index int, mainText, secondaryText string, shortcut rune) {
	// Preemptively load more when approaching the end (last 3 items)
	threshold := len(s.posts) - 3
	if threshold < 0 {
		threshold = 0
	}
	if index >= threshold && !s.loadingMore && s.offset < s.totalPosts {
		s.loadMore()
	}

	if index < 0 || index >= len(s.posts) {
		return
	}
	post := s.posts[index]
	s.selectedPostID = post.ID

	if cached, ok := s.getCachedComments(post.ID); ok {
		s.renderDetailLoading(post)
		s.renderDetailComments(post, cached)
		return
	}

	s.renderDetailLoading(post)

	go func(p models.Post, selectedID int) {
		comments, err := api.FetchTopComments(p, 5)
		if err != nil {
			log.Printf("Failed to fetch comments: %v", err)
			s.app.QueueUpdateDraw(func() {
				if s.selectedPostID == selectedID {
					s.renderDetailError(p, err)
				}
			})
			return
		}
		s.setCachedComments(p.ID, comments)
		s.app.QueueUpdateDraw(func() {
			if s.selectedPostID == selectedID {
				s.renderDetailComments(p, comments)
			}
		})
	}(post, post.ID)
}

func (s *appState) renderDetailLoading(post models.Post) {
	s.detailView.SetText(s.detailHeader(post) + "\n\n[dim]Loading comments...")
}

func (s *appState) renderDetailError(post models.Post, err error) {
	s.detailView.SetText(s.detailHeader(post) + fmt.Sprintf("\n\n[red]Error loading comments: %v", err))
}

func (s *appState) renderDetailComments(post models.Post, comments []models.Comment) {
	text := s.detailHeader(post) + "\n\n"
	if len(comments) > 0 {
		text += "[cyan::b]Top Comments:\n\n"
		for i, comment := range comments {
			if i >= 5 {
				break
			}
			ct := utils.StripHTML(comment.Text)
			if len(ct) > 300 {
				ct = ct[:300] + "..."
			}
			text += fmt.Sprintf("[white]%d. [gray::i]%s[white:-]\n%s\n\n", i+1, comment.Author, ct)
		}
	} else {
		text += "[dim]No comments yet\n\n"
	}
	text += "[dim]Press 'o' or Enter to open in browser"
	s.detailView.SetText(text)
}

func (s *appState) detailHeader(post models.Post) string {
	url := post.URL
	if url == "" {
		url = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", post.ID)
	}
	return fmt.Sprintf(
		"[yellow::b]Title:[white:-] %s\n\n"+
			"[dodgerblue::b]Author:[white:-] %s\n"+
			"[orange::b]Score:[white:-] %d points\n"+
			"[green::b]Comments:[white:-] %d\n"+
			"[purple::b]Type:[white:-] %s\n"+
			"[gray::b]Time:[white:-] %s\n\n"+
			"[gray::b]URL:[white:-]\n%s",
		post.Title,
		post.Author,
		post.Score,
		post.Comments,
		post.Type,
		utils.FormatTimeAgo(post.Time),
		url,
	)
}

func (s *appState) setupInputCapture() {
	s.list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Block navigation while loading more to prevent race conditions
		if s.loadingMore {
			switch event.Key() {
			case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn, tcell.KeyHome, tcell.KeyEnd:
				return nil
			}
			switch event.Rune() {
			case 'n':
				return nil
			}
		}

		switch event.Rune() {
		case 'q':
			s.app.Stop()
			return nil
		case 'o':
			s.openCurrentURL()
			return nil
		case 'h':
			s.openCurrentHN()
			return nil
		case 'r':
			s.refresh()
			return nil
		case 'n':
			s.loadMore()
			return nil
		case '1':
			s.switchFeed(api.FeedTop)
			return nil
		case '2':
			s.switchFeed(api.FeedNew)
			return nil
		case '3':
			s.switchFeed(api.FeedBest)
			return nil
		case '4':
			s.switchFeed(api.FeedAsk)
			return nil
		case '5':
			s.switchFeed(api.FeedShow)
			return nil
		case '6':
			s.switchFeed(api.FeedJob)
			return nil
		}

		switch event.Key() {
		case tcell.KeyEnter:
			s.openCurrentURL()
			return nil
		case tcell.KeyEscape:
			s.app.Stop()
			return nil
		}

		return event
	})
}

func (s *appState) currentPost() *models.Post {
	idx := s.list.GetCurrentItem()
	if idx >= 0 && idx < len(s.posts) {
		return &s.posts[idx]
	}
	return nil
}

func (s *appState) openCurrentURL() {
	post := s.currentPost()
	if post == nil {
		return
	}
	url := post.URL
	if url == "" {
		url = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", post.ID)
	}
	if err := utils.OpenInBrowser(url); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

func (s *appState) openCurrentHN() {
	post := s.currentPost()
	if post == nil {
		return
	}
	hnURL := fmt.Sprintf("https://news.ycombinator.com/item?id=%d", post.ID)
	if err := utils.OpenInBrowser(hnURL); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

func (s *appState) refresh() {
	s.offset = 0
	s.clearCache()
	go s.loadFeed(s.feed, 0)
}

func (s *appState) switchFeed(feed string) {
	if feed == s.feed {
		return
	}
	s.offset = 0
	s.clearCache()
	s.feed = feed
	go s.loadFeed(feed, 0)
}

func (s *appState) loadFeed(feed string, offset int) {
	s.app.QueueUpdateDraw(func() {
		s.list.SetTitle(" ⏳ Loading... ")
	})
	posts, total, err := api.FetchPosts(feed, offset, s.pageSize)
	s.app.QueueUpdateDraw(func() {
		if err != nil {
			s.detailView.SetText(fmt.Sprintf("[red]Error loading feed: %v", err))
			s.list.SetTitle(feedTitle(feed))
			return
		}
		s.posts = posts
		s.offset = len(posts)
		s.totalPosts = total
		s.populateList()
		if len(posts) > 0 {
			s.list.SetCurrentItem(0)
		}
		s.list.SetTitle(feedTitle(feed))
		s.updateStatusBar()
	})
}

func (s *appState) loadMore() {
	if s.loadingMore || s.offset >= s.totalPosts {
		return
	}
	s.loadingMore = true
	s.updateStatusBar()
	go func() {
		posts, total, err := api.FetchPosts(s.feed, s.offset, s.pageSize)
		s.app.QueueUpdateDraw(func() {
			s.loadingMore = false
			if err != nil {
				s.updateStatusBar()
				return
			}
			if len(posts) == 0 {
				s.totalPosts = total
				s.updateStatusBar()
				return
			}
			s.appendPosts(posts)
			s.offset += len(posts)
			s.totalPosts = total
			s.updateStatusBar()
		})
	}()
}

func (s *appState) updateStatusBar() {
	var text string
	if s.loadingMore {
		text = fmt.Sprintf(
			"[yellow]⏳ Loading more stories... (%d/%d)  |  [yellow]q[white]/[yellow]Esc[white]: Quit",
			s.offset, s.totalPosts,
		)
	} else {
		text = fmt.Sprintf(
			"[yellow]o[white]/[yellow]Enter[white]: Open  |  [yellow]h[white]: HN  |  [yellow]n[white]: Next (%d/%d)  |  [yellow]r[white]: Refresh  |  [yellow]1-6[white]: Feed  |  [yellow]↑↓[white]: Nav  |  [yellow]q[white]/[yellow]Esc[white]: Quit",
			s.offset, s.totalPosts,
		)
	}
	s.statusBar.SetText(text)
}

func (s *appState) getCachedComments(postID int) ([]models.Comment, bool) {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	c, ok := s.cache[postID]
	return c, ok
}

func (s *appState) setCachedComments(postID int, comments []models.Comment) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cache[postID] = comments
}

func (s *appState) clearCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cache = make(map[int][]models.Comment)
}

func feedTitle(feed string) string {
	switch feed {
	case api.FeedTop:
		return " 🔥 Top Stories "
	case api.FeedNew:
		return " 🆕 New Stories "
	case api.FeedBest:
		return " ⭐ Best Stories "
	case api.FeedAsk:
		return " ❓ Ask HN "
	case api.FeedShow:
		return " 🚀 Show HN "
	case api.FeedJob:
		return " 💼 Jobs "
	default:
		return " 🔥 Hacker News "
	}
}

func ShowLoadingScreen(app *tview.Application) {
	loadingText := tview.NewTextView().
		SetText("[yellow::b]Loading Hacker News posts...\n\n[white]Please wait...").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	loadingText.SetBorder(true).
		SetTitle(" 🔥 Hacker News ").
		SetTitleColor(tcell.ColorOrange).
		SetBorderColor(tcell.ColorOrange)

	app.SetRoot(loadingText, true)
}
