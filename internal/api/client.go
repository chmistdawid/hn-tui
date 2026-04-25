package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/chmistdawid/hn-tui/internal/models"
)

const baseURL = "https://hacker-news.firebaseio.com/v0"

var httpClient = &http.Client{Timeout: 10 * time.Second}

const (
	FeedTop  = "topstories"
	FeedNew  = "newstories"
	FeedBest = "beststories"
	FeedAsk  = "askstories"
	FeedShow = "showstories"
	FeedJob  = "jobstories"
)

func fetchJSON(ctx context.Context, url string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, target)
}

func FetchPost(ctx context.Context, postID string) (*models.Post, error) {
	var post models.Post
	url := fmt.Sprintf("%s/item/%s.json", baseURL, postID)
	if err := fetchJSON(ctx, url, &post); err != nil {
		return nil, err
	}
	return &post, nil
}

func FetchPosts(ctx context.Context, feed string, offset, limit int) ([]models.Post, int, error) {
	var postIDs []int
	url := fmt.Sprintf("%s/%s.json", baseURL, feed)
	if err := fetchJSON(ctx, url, &postIDs); err != nil {
		return nil, 0, err
	}

	total := len(postIDs)

	if offset >= total {
		return []models.Post{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}
	postIDs = postIDs[offset:end]

	postList := make([]models.Post, len(postIDs))
	var wg sync.WaitGroup

	for i, id := range postIDs {
		wg.Add(1)
		go func(index int, postID int) {
			defer wg.Done()

			post, err := FetchPost(ctx, fmt.Sprintf("%d", postID))
			if err != nil {
				return
			}

			postList[index] = *post
		}(i, id)
	}

	wg.Wait()

	var result []models.Post
	for _, p := range postList {
		if p.ID != 0 {
			result = append(result, p)
		}
	}

	if len(result) == 0 {
		return nil, total, fmt.Errorf("all %d posts failed to load", len(postIDs))
	}

	return result, total, nil
}

func FetchComment(ctx context.Context, commentID int) (*models.Comment, error) {
	var comment models.Comment
	url := fmt.Sprintf("%s/item/%d.json", baseURL, commentID)
	if err := fetchJSON(ctx, url, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

func FetchTopComments(ctx context.Context, post models.Post, limit int) ([]models.Comment, error) {
	if len(post.Kids) == 0 {
		return []models.Comment{}, nil
	}

	kidIDs := post.Kids
	if limit > 0 && limit < len(kidIDs) {
		kidIDs = kidIDs[:limit]
	}

	comments := make([]models.Comment, len(kidIDs))
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, 10)

	for i, kidID := range kidIDs {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(index int, id int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			comment, err := FetchComment(ctx, id)
			if err != nil || comment.Deleted || comment.Dead {
				return
			}
			comments[index] = *comment
		}(i, kidID)
	}

	wg.Wait()

	var result []models.Comment
	for _, c := range comments {
		if c.ID != 0 {
			result = append(result, c)
		}
	}

	return result, nil
}
