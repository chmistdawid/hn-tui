package ui

import (
	"fmt"
	"log"

	"github.com/chmistdawid/hn-tui/internal/api"
	"github.com/chmistdawid/hn-tui/internal/models"
	"github.com/chmistdawid/hn-tui/internal/utils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func SetupMainUI(app *tview.Application, posts []models.Post) {
	list := tview.NewList()
	list.SetBorder(true).
		SetTitle(" 🔥 Hacker News - Top Stories ").
		SetTitleColor(tcell.ColorOrange).
		SetBorderColor(tcell.ColorOrange).
		SetBorderPadding(0, 0, 1, 1)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSecondaryTextColor(tcell.ColorGray)
	list.SetSelectedTextColor(tcell.ColorWhite)
	list.SetSelectedBackgroundColor(tcell.ColorDarkCyan)
	list.SetHighlightFullLine(true)
	list.ShowSecondaryText(true)

	detailView := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true)
	detailView.SetBorder(true).
		SetTitle(" 📝 Details ").
		SetTitleColor(tcell.ColorDodgerBlue).
		SetBorderColor(tcell.ColorDodgerBlue).
		SetBorderPadding(0, 0, 1, 1)

	for i, post := range posts {
		title := fmt.Sprintf("[white::b]%d. [yellow]▲ %d [white]%s", i+1, post.Score, post.Title)
		secondary := fmt.Sprintf("[gray]by [::i]%s[gray:-] | [dodgerblue]%d comments", post.Author, post.Comments)
		list.AddItem(title, secondary, 0, nil)
	}

	list.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(posts) {
			post := posts[index]
			url := post.URL
			if url == "" {
				url = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", post.ID)
			}

			detailText := fmt.Sprintf(
				"[yellow::b]Title:[white:-] %s\n\n"+
					"[dodgerblue::b]Author:[white:-] %s\n"+
					"[orange::b]Score:[white:-] %d points\n"+
					"[green::b]Comments:[white:-] %d\n"+
					"[purple::b]Type:[white:-] %s\n\n"+
					"[gray::b]URL:[white:-]\n%s\n\n"+
					"[dim]Loading comments...",
				post.Title,
				post.Author,
				post.Score,
				post.Comments,
				post.Type,
				url,
			)
			detailView.SetText(detailText)

			go func(p models.Post) {
				comments, err := api.FetchTopComments(p, 5)
				if err != nil {
					log.Printf("Failed to fetch comments: %v", err)
					return
				}

				commentText := fmt.Sprintf(
					"[yellow::b]Title:[white:-] %s\n\n"+
						"[dodgerblue::b]Author:[white:-] %s\n"+
						"[orange::b]Score:[white:-] %d points\n"+
						"[green::b]Comments:[white:-] %d\n"+
						"[purple::b]Type:[white:-] %s\n\n"+
						"[gray::b]URL:[white:-]\n%s\n\n",
					p.Title,
					p.Author,
					p.Score,
					p.Comments,
					p.Type,
					url,
				)

				if len(comments) > 0 {
					commentText += "[cyan::b]Top Comments:\n\n"
					for i, comment := range comments {
						if i >= 5 {
							break
						}
						text := utils.StripHTML(comment.Text)
						if len(text) > 200 {
							text = text[:200] + "..."
						}
						commentText += fmt.Sprintf("[white]%d. [gray::i]%s[white:-]\n%s\n\n", i+1, comment.Author, text)
					}
				} else {
					commentText += "[dim]No comments yet\n\n"
				}

				commentText += "[dim]Press 'o' or Enter to open in browser"

				app.QueueUpdateDraw(func() {
					detailView.SetText(commentText)
				})
			}(post)
		}
	})

	if len(posts) > 0 {
		list.SetCurrentItem(0)
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'h' {
			currentIndex := list.GetCurrentItem()
			if currentIndex >= 0 && currentIndex < len(posts) {
				hnURL := fmt.Sprintf("https://news.ycombinator.com/item?id=%d", posts[currentIndex].ID)
				if err := utils.OpenInBrowser(hnURL); err != nil {
					log.Printf("Failed to open browser: %v", err)
				}
			}
			return nil
		}
		if event.Key() == tcell.KeyEnter || event.Rune() == 'o' {
			currentIndex := list.GetCurrentItem()
			if currentIndex >= 0 && currentIndex < len(posts) {
				url := posts[currentIndex].URL
				if url == "" {
					url = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", posts[currentIndex].ID)
				}
				if err := utils.OpenInBrowser(url); err != nil {
					log.Printf("Failed to open browser: %v", err)
				}
			}
			return nil
		}
		if event.Rune() == 'q' || event.Key() == tcell.KeyEscape {
			app.Stop()
			return nil
		}
		return event
	})

	helpBar := tview.NewFlex().
		SetDirection(tview.FlexRow)

	helpText := tview.NewTextView().
		SetText("[yellow]o[white]/[yellow]Enter[white]: Open link  |  [yellow]h[white]: Open HN comments  |  [yellow]↑↓[white]: Navigate  |  [yellow]q[white]/[yellow]Esc[white]: Quit").
		SetTextAlign(tview.AlignCenter).
		SetMaxLines(1).
		SetDynamicColors(true).
		SetTextColor(tcell.ColorWhite)

	helpBar.AddItem(helpText, 1, 1, false)

	mainContent := tview.NewFlex().
		AddItem(list, 0, 2, true).
		AddItem(detailView, 0, 1, false)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(mainContent, 0, 1, true).
		AddItem(helpBar, 1, 1, false)

	app.SetRoot(flex, true).SetFocus(list)
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
