package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor   = lipgloss.Color("#7D56F4") // Violet
	accentColor    = lipgloss.Color("#04B575") // Emerald Green
	warningColor   = lipgloss.Color("#FF4C54") // Coral Red
	textColor      = lipgloss.Color("#C1C6E2") // Pastel Gray
	dimColor       = lipgloss.Color("#62688F") // Muted Blue-Gray
	highlightColor = lipgloss.Color("#EE6FF8") // Pink

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Padding(0, 2).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Underline(true).
			MarginBottom(1)

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dimColor).
			Padding(1, 2).
			MarginBottom(1)

	activeCardStyle = cardStyle.Copy().
			BorderForeground(primaryColor)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(highlightColor).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(textColor)

	positiveAmountStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)

	negativeAmountStyle = lipgloss.NewStyle().
				Foreground(warningColor).
				Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)
)
