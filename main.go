package main

import (
	"fmt"
	"os"

	"billshare/pkg/storage"
	"billshare/pkg/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	dbPath := "billshare.json"

	store, err := storage.NewJSONStore(dbPath)
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(tui.NewModel(store), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
