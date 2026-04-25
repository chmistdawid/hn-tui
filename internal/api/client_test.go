package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/chmistdawid/hn-tui/internal/models"
)

func TestFetchPost(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	// Save original and override httpClient
	originalClient := httpClient
	httpClient = &http.Client{Timeout: 5 * time.Second}
	defer func() { httpClient = originalClient }()

	expectedPost := models.Post{
		ID:     123,
		Author: "testuser",
		Score:  42,
		Title:  "Test Post",
		Type:   "story",
		URL:    "http://example.com",
	}

	mux.HandleFunc("/item/123.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedPost)
	})

	mux.HandleFunc("/item/999.json", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	t.Run("successful fetch", func(t *testing.T) {
		ctx := context.Background()
		post, err := fetchPostWithBaseURL(ctx, server.URL, "123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if post.ID != expectedPost.ID {
			t.Errorf("expected ID %d, got %d", expectedPost.ID, post.ID)
		}
		if post.Author != expectedPost.Author {
			t.Errorf("expected Author %q, got %q", expectedPost.Author, post.Author)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		_, err := fetchPostWithBaseURL(ctx, server.URL, "999")
		if err == nil {
			t.Error("expected error for 404 response")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		mux.HandleFunc("/item/456.json", func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			json.NewEncoder(w).Encode(expectedPost)
		})

		_, err := fetchPostWithBaseURL(ctx, server.URL, "456")
		if err == nil {
			t.Error("expected error for cancelled context")
		}
	})
}

func TestFetchPosts_OrderPreservation(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	originalClient := httpClient
	httpClient = &http.Client{Timeout: 5 * time.Second}
	defer func() { httpClient = originalClient }()

	// IDs we'll fetch
	storyIDs := []int{100, 200, 300}

	mux.HandleFunc("/topstories.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(storyIDs)
	})

	// Each item endpoint returns a post with ID matching the request
	for _, id := range storyIDs {
		id := id // capture for closure
		mux.HandleFunc(fmt.Sprintf("/item/%d.json", id), func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(models.Post{
				ID:     id,
				Title:  fmt.Sprintf("Post %d", id),
				Author: "user",
			})
		})
	}

	ctx := context.Background()
	posts, total, err := fetchPostsWithBaseURL(ctx, server.URL, FeedTop, 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != len(storyIDs) {
		t.Errorf("expected total %d, got %d", len(storyIDs), total)
	}

	if len(posts) != len(storyIDs) {
		t.Fatalf("expected %d posts, got %d", len(storyIDs), len(posts))
	}

	// Check order is preserved
	for i, expectedID := range storyIDs {
		if posts[i].ID != expectedID {
			t.Errorf("post at index %d: expected ID %d, got %d", i, expectedID, posts[i].ID)
		}
	}
}

func TestFetchPosts_PartialFailure(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	originalClient := httpClient
	httpClient = &http.Client{Timeout: 5 * time.Second}
	defer func() { httpClient = originalClient }()

	storyIDs := []int{100, 200, 300}

	mux.HandleFunc("/topstories.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(storyIDs)
	})

	// First two succeed, third fails
	mux.HandleFunc("/item/100.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(models.Post{ID: 100, Title: "Post 100"})
	})
	mux.HandleFunc("/item/200.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(models.Post{ID: 200, Title: "Post 200"})
	})
	mux.HandleFunc("/item/300.json", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	ctx := context.Background()
	posts, _, err := fetchPostsWithBaseURL(ctx, server.URL, FeedTop, 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(posts) != 2 {
		t.Errorf("expected 2 posts (partial failure), got %d", len(posts))
	}

	// Check order: 100 should be first, 200 second
	if len(posts) >= 1 && posts[0].ID != 100 {
		t.Errorf("expected first post ID 100, got %d", posts[0].ID)
	}
	if len(posts) >= 2 && posts[1].ID != 200 {
		t.Errorf("expected second post ID 200, got %d", posts[1].ID)
	}
}

func TestFetchPosts_AllFail(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	originalClient := httpClient
	httpClient = &http.Client{Timeout: 5 * time.Second}
	defer func() { httpClient = originalClient }()

	storyIDs := []int{100, 200}

	mux.HandleFunc("/topstories.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(storyIDs)
	})

	// All fail
	for _, id := range storyIDs {
		id := id
		mux.HandleFunc(fmt.Sprintf("/item/%d.json", id), func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
	}

	ctx := context.Background()
	_, _, err := fetchPostsWithBaseURL(ctx, server.URL, FeedTop, 0, 10)
	if err == nil {
		t.Error("expected error when all posts fail")
	}
}

func TestFetchTopComments_OrderPreservation(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	originalClient := httpClient
	httpClient = &http.Client{Timeout: 5 * time.Second}
	defer func() { httpClient = originalClient }()

	kids := []int{10, 20, 30}

	// Each comment endpoint returns a comment with ID matching the request
	for _, id := range kids {
		id := id
		mux.HandleFunc(fmt.Sprintf("/item/%d.json", id), func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(models.Comment{
				ID:     id,
				Text:   fmt.Sprintf("Comment %d", id),
				Author: "user",
			})
		})
	}

	post := models.Post{
		ID:   1,
		Kids: kids,
	}

	ctx := context.Background()
	comments, err := fetchTopCommentsWithBaseURL(ctx, server.URL, post, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(comments) != len(kids) {
		t.Fatalf("expected %d comments, got %d", len(kids), len(comments))
	}

	// Check order is preserved
	for i, expectedID := range kids {
		if comments[i].ID != expectedID {
			t.Errorf("comment at index %d: expected ID %d, got %d", i, expectedID, comments[i].ID)
		}
	}
}

// Helper functions that allow injecting a custom base URL for testing

func fetchJSONWithBaseURL(ctx context.Context, base, path string, target interface{}) error {
	url := base + path
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

func fetchPostWithBaseURL(ctx context.Context, base, postID string) (*models.Post, error) {
	var post models.Post
	if err := fetchJSONWithBaseURL(ctx, base, "/item/"+postID+".json", &post); err != nil {
		return nil, err
	}
	return &post, nil
}

func fetchPostsWithBaseURL(ctx context.Context, base, feed string, offset, limit int) ([]models.Post, int, error) {
	var postIDs []int
	if err := fetchJSONWithBaseURL(ctx, base, "/"+feed+".json", &postIDs); err != nil {
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
			post, err := fetchPostWithBaseURL(ctx, base, fmt.Sprintf("%d", postID))
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

func fetchCommentWithBaseURL(ctx context.Context, base string, commentID int) (*models.Comment, error) {
	var comment models.Comment
	if err := fetchJSONWithBaseURL(ctx, base, fmt.Sprintf("/item/%d.json", commentID), &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

func fetchTopCommentsWithBaseURL(ctx context.Context, base string, post models.Post, limit int) ([]models.Comment, error) {
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

			comment, err := fetchCommentWithBaseURL(ctx, base, id)
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
