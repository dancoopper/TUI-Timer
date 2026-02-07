package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// STYLES
var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle  = focusedStyle.Copy()
	noStyle      = lipgloss.NewStyle()
	helpStyle    = blurredStyle.Copy()

	focusedButton = focusedStyle.Copy().Render("[ %s ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("%s"))
)

type model struct {
	textInput  textinput.Model
	duration   time.Duration
	remaining  time.Duration
	running    bool
	focusIndex int // 0: input, 1: start, 2: stop, 3: quit
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "10s (e.g. 5m, 1h30m)"
	ti.Focus()
	ti.CharLimit = 20
	ti.Width = 30

	return model{
		textInput:  ti,
		focusIndex: 0,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

type tickMsg time.Time

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab", "shift+tab", "left", "right", "up", "down":
			s := msg.String()

			switch s {
			case "tab":
				m.focusIndex++
				break
			case "shift+tab":
				m.focusIndex--
				break
			case "left":
				m.focusIndex--
				break
			case "right":
				m.focusIndex++
				break
			case "up":
				if m.focusIndex > 0 {
					m.focusIndex = 0
					break
				}
				m.focusIndex--
				break
			case "down":
				if m.focusIndex > 0 {

					break
				}
				m.focusIndex++
				break
			}

			if m.focusIndex > 3 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = 3
			}

			// Handle Input Focus
			if m.focusIndex == 0 {
				cmd = m.textInput.Focus()
			} else {
				m.textInput.Blur()
			}
			return m, cmd

		case "enter":
			if m.focusIndex == 0 { // Input field
				// Treat Enter in input field as "Start" if valid, or just move focus?
				// User might expect to submit. Let's try to start.
				parsed, err := time.ParseDuration(m.textInput.Value())
				if err == nil && parsed > 0 {
					m.duration = parsed
					m.remaining = parsed
					m.running = true
					m.textInput.Blur()
					m.focusIndex = 2 // Move focus to Stop
					return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
						return tickMsg(t)
					})
				}
			} else if m.focusIndex == 1 { // Start Button
				parsed, err := time.ParseDuration(m.textInput.Value())
				if err == nil && parsed > 0 {
					m.duration = parsed
					m.remaining = parsed
					m.running = true
					return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
						return tickMsg(t)
					})
				}
			} else if m.focusIndex == 2 { // Stop Button
				m.running = false
			} else if m.focusIndex == 3 { // Quit Button
				return m, tea.Quit
			}
		}

	case tickMsg:
		if m.running && m.remaining > 0 {
			m.remaining -= time.Second
			if m.remaining <= 0 {
				m.running = false
				m.remaining = 0
				return m, nil // or maybe play a sound/notify
			}
			return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
				return tickMsg(t)
			})
		}
	}

	// Update text input only if focused
	if m.focusIndex == 0 {
		m.textInput, cmd = m.textInput.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	var s strings.Builder

	s.WriteString("\n  Timer TUI\n\n")

	// Input
	s.WriteString("  Duration: ")
	s.WriteString(m.textInput.View())
	s.WriteString("\n\n")

	// Timer Display
	if m.remaining > 0 || m.running {
		s.WriteString(fmt.Sprintf("  Time Remaining: %s\n\n", m.remaining.Round(time.Second)))
	} else {
		s.WriteString("  Time Remaining: 0s\n\n")
	}

	// Buttons
	startButton := fmt.Sprintf("[ %s ]", "Start")
	if m.focusIndex == 1 {
		startButton = fmt.Sprintf(focusedButton, "Start")
	} else {
		startButton = fmt.Sprintf(blurredButton, "Start")
	}

	stopButton := fmt.Sprintf("[ %s ]", "Stop")
	if m.focusIndex == 2 {
		stopButton = fmt.Sprintf(focusedButton, "Stop")
	} else {
		stopButton = fmt.Sprintf(blurredButton, "Stop")
	}

	quitButton := fmt.Sprintf("[ %s ]", "Quit")
	if m.focusIndex == 3 {
		quitButton = fmt.Sprintf(focusedButton, "Quit")
	} else {
		quitButton = fmt.Sprintf(blurredButton, "Quit")
	}

	s.WriteString(fmt.Sprintf("  %s  %s  %s\n\n", startButton, stopButton, quitButton))

	s.WriteString(helpStyle.Render("  (Tab to navigate, Enter to select)\n"))

	return s.String()
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
