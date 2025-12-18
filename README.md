# hn-tui

A terminal user interface for browsing Hacker News built with Go and tview.

## Features

- Browse top stories from Hacker News
- View post details including title, author, score, and comments
- Read top comments for each post
- Open posts in browser with a keyboard shortcut
- Fast concurrent loading of posts and comments
- Clean and colorful TUI interface

## Installation

```bash
go install github.com/chmistdawid/hn-tui@0.0.1
```

Or clone and build:

```bash
git clone https://github.com/chmistdawid/hn-tui.git
cd hn-tui
go build -o hn-tui .
```

## Usage

Run the application:

```bash
./hn-tui
```

### Keyboard Shortcuts

- `o` or `Enter` - Open selected post in browser
- `h` - Open the HN comments page in browser
- `↑ `/ `↓ ` - Navigate through posts
- `q` or `Esc` - Quit application

## Project Structure

```
hn-tui/
├── main.go               # Application entry point
├── internal/
│   ├── api/
│   │   └── client.go     # Hacker News API client
│   ├── models/
│   │   └── models.go     # Data models
│   ├── ui/
│   │   └── ui.go         # TUI interface
│   └── utils/
│       └── utils.go      # Utility functions
├── go.mod
└── README.md
```

## License

MIT
