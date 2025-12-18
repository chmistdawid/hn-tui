package main

import (
	"log"

	"github.com/chmistdawid/hn-tui/internal/api"
	"github.com/chmistdawid/hn-tui/internal/ui"
	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()

	ui.ShowLoadingScreen(app)

	go func() {
		posts, err := api.FetchTopPosts(30)
		if err != nil {
			app.Stop()
			log.Fatalf("Could not download HN posts: %v", err)
		}

		app.QueueUpdateDraw(func() {
			ui.SetupMainUI(app, posts)
		})
	}()

	if err := app.EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
