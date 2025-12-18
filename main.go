package main

import "fmt"

func main() {
	fmt.Println("This directory is not an install target.")
	fmt.Println("Install the binary with:")
	fmt.Println("  go install github.com/chmistdawid/hn-tui/cmd/hn-tui@latest")
}
