package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/chmistdawid/hn-tui/internal/models"
)

const baseURL = "https://hacker-news.firebaseio.com/v0"

func FetchPost(postID string) (*models.Post, error) {
	url := fmt.Sprintf("%s/item/%s.json", baseURL, postID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var post models.Post
	err = json.Unmarshal(body, &post)
	if err != nil {
		return nil, err
	}

	return &post, nil
}

func FetchTopPosts(limit int) ([]models.Post, error) {
	resp, err := http.Get(fmt.Sprintf("%s/topstories.json", baseURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var postIDs []int
	err = json.Unmarshal(body, &postIDs)
	if err != nil {
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
	url := fmt.Sprintf("%s/item/%d.json", baseURL, commentID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var comment models.Comment
	err = json.Unmarshal(body, &comment)
	if err != nil {
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

	comments := make([]models.Comment, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, kidID := range kidIDs {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
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
