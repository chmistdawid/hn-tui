package models

type Post struct {
	ID       int    `json:"id"`
	Author   string `json:"by"`
	Score    int    `json:"score"`
	Title    string `json:"title"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Comments int    `json:"descendants"`
	Time     int64  `json:"time"`
	Kids     []int  `json:"kids"`
}

type Comment struct {
	ID      int    `json:"id"`
	Author  string `json:"by"`
	Text    string `json:"text"`
	Kids    []int  `json:"kids"`
	Deleted bool   `json:"deleted"`
	Dead    bool   `json:"dead"`
}
