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

type Focus int

const (
	INPUT = Focus(0)
	START = Focus(1)
	STOP  = Focus(2)
	RESET = Focus(3)
	QUIT  = Focus(4)
)

type model struct {
	textInput  textinput.Model
	duration   time.Duration
	remaining  time.Duration
	running    bool
	focusIndex Focus // 0: input, 1: start, 2: stop, 3: quit
	focusState Focus // what the last thing that was focused? maybe a bad Idea but we will see
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "10s (e.g. 5m, 1h30m)"
	ti.Focus()
	ti.CharLimit = 20
	ti.Width = 30

	return model{
		textInput:  ti,
		focusIndex: INPUT,
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

			case "shift+tab":
				m.focusIndex--

			case "left":

				if m.focusIndex == 0 {
					m.focusIndex = START
					m.focusState = START
					break
				}

				if m.focusIndex == START {
					m.focusIndex = QUIT
					m.focusState = QUIT
					break
				}

				m.focusIndex--
				m.focusState = m.focusIndex // get the last thing we focused on so we can have a sort of memory of what the last state was before we changed it

			case "right":
				if m.focusIndex == INPUT {
					m.focusIndex = QUIT
					m.focusState = QUIT
					break
				}
				if m.focusIndex == QUIT {
					m.focusIndex = START
					m.focusState = START
					break
				}

				m.focusIndex++
				m.focusState = m.focusIndex // get the last thing we focused on so we can have a sort of memory of what the last state was before we changed it

			case "up":
				if m.focusIndex > INPUT { // if in input field, move to start button
					m.focusIndex = INPUT
					break
				}
				//m.focusIndex--

			case "down":

				if m.focusIndex == INPUT {
					m.focusIndex = m.focusState
				}

				if m.focusIndex > INPUT { // if in the buttons part and user press down, app should do nothing
					break
				}
				m.focusIndex++

			}

			if m.focusIndex > QUIT {
				m.focusIndex = INPUT
			} else if m.focusIndex < INPUT {
				m.focusIndex = QUIT
			}

			// Handle Input Focus
			if m.focusIndex == INPUT {
				cmd = m.textInput.Focus()
			} else {
				m.textInput.Blur()
			}
			return m, cmd

		case "enter":

			if m.focusIndex == INPUT { // Input field
				// Treat Enter in input field as "Start" if valid, or just move focus?
				// User might expect to submit. Let's try to start.
				parsed, err := time.ParseDuration(m.textInput.Value())
				if err == nil && parsed > 0 {
					m.duration = parsed
					m.remaining = parsed
					m.running = true
					m.textInput.SetValue("")
					m.textInput.Blur()
					m.focusIndex = STOP // Move focus to Stop
					return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
						return tickMsg(t)
					})
				}
			} else if m.focusIndex == START { // Start Button

				parsed, err := time.ParseDuration(m.textInput.Value())

				if err != nil {
					parsed = m.remaining
				}

				if parsed > 0 {
					m.duration = parsed
					m.remaining = parsed
					m.running = true
					m.textInput.SetValue("")
					return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
						return tickMsg(t)
					})
				}

			} else if m.focusIndex == STOP { // Stop Button
				m.running = false
			} else if m.focusIndex == RESET { // Reset Button
				m.duration = 0
				m.remaining = 0
				m.running = false
				m.textInput.SetValue("")
				//m.focusIndex = 0
			} else if m.focusIndex == 4 { // Quit Button
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
	if m.focusIndex == INPUT {
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
	if m.focusIndex == START {
		startButton = fmt.Sprintf(focusedButton, "Start")
	} else {
		startButton = fmt.Sprintf(blurredButton, "Start")
	}

	stopButton := fmt.Sprintf("[ %s ]", "Stop")
	if m.focusIndex == STOP {
		stopButton = fmt.Sprintf(focusedButton, "Stop")
	} else {
		stopButton = fmt.Sprintf(blurredButton, "Stop")
	}

	resetButton := fmt.Sprintf("[ %s ]", "Reset")
	if m.focusIndex == RESET {
		resetButton = fmt.Sprintf(focusedButton, "Reset")
	} else {
		resetButton = fmt.Sprintf(blurredButton, "Reset")
	}

	quitButton := fmt.Sprintf("[ %s ]", "Quit")
	if m.focusIndex == QUIT {
		quitButton = fmt.Sprintf(focusedButton, "Quit")
	} else {
		quitButton = fmt.Sprintf(blurredButton, "Quit")
	}

	s.WriteString(fmt.Sprintf("  %s  %s  %s  %s\n\n", startButton, stopButton, resetButton, quitButton))

	s.WriteString(helpStyle.Render("  (Tab to navigate, Enter to select)\n"))

	return s.String()
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
