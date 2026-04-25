package main

import (
	"context"
	"log"

	"github.com/chmistdawid/hn-tui/internal/api"
	"github.com/chmistdawid/hn-tui/internal/ui"
	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()

	ui.ShowLoadingScreen(app)

	go func() {
		posts, total, err := api.FetchPosts(context.Background(), api.FeedTop, 0, 30)
		if err != nil {
			app.Stop()
			log.Fatalf("Could not download HN posts: %v", err)
		}

		app.QueueUpdateDraw(func() {
			ui.SetupMainUI(app, posts, total)
		})
	}()

	if err := app.EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
