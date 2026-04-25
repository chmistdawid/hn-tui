package api

import (
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

func fetchJSON(url string, target interface{}) error {
	resp, err := httpClient.Get(url)
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

func FetchPost(postID string) (*models.Post, error) {
	var post models.Post
	url := fmt.Sprintf("%s/item/%s.json", baseURL, postID)
	if err := fetchJSON(url, &post); err != nil {
		return nil, err
	}
	return &post, nil
}

func FetchPosts(feed string, limit int) ([]models.Post, error) {
	var postIDs []int
	url := fmt.Sprintf("%s/%s.json", baseURL, feed)
	if err := fetchJSON(url, &postIDs); err != nil {
		return nil, err
	}

	if limit > 0 && limit < len(postIDs) {
		postIDs = postIDs[:limit]
	}

	postList := make([]models.Post, len(postIDs))
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make(chan error, len(postIDs))

	for i, id := range postIDs {
		wg.Add(1)
		go func(index int, postID int) {
			defer wg.Done()

			post, err := FetchPost(fmt.Sprintf("%d", postID))
			if err != nil {
				errors <- err
				return
			}

			mu.Lock()
			postList[index] = *post
			mu.Unlock()
		}(i, id)
	}

	wg.Wait()
	close(errors)

	if len(errors) > 0 {
		return nil, <-errors
	}

	return postList, nil
}

func FetchComment(commentID int) (*models.Comment, error) {
	var comment models.Comment
	url := fmt.Sprintf("%s/item/%d.json", baseURL, commentID)
	if err := fetchJSON(url, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

func FetchTopComments(post models.Post, limit int) ([]models.Comment, error) {
	if len(post.Kids) == 0 {
		return []models.Comment{}, nil
	}

	kidIDs := post.Kids
	if limit > 0 && limit < len(kidIDs) {
		kidIDs = kidIDs[:limit]
	}

	comments := make([]models.Comment, 0, len(kidIDs))
	var mu sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, 10)

	for _, kidID := range kidIDs {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(id int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			comment, err := FetchComment(id)
			if err != nil || comment.Deleted || comment.Dead {
				return
			}
			mu.Lock()
			comments = append(comments, *comment)
			mu.Unlock()
		}(kidID)
	}

	wg.Wait()
	return comments, nil
}
