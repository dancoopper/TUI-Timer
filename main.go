package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

	// Animation styles
	alarmStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true) // Red bold
)

type Focus int

const (
	INPUT = Focus(0)
	ADD   = Focus(1)
	START = Focus(2)
	STOP  = Focus(3)
	RESET = Focus(4)
	QUIT  = Focus(5)
)

type Timer struct {
	ID        int
	Duration  time.Duration
	Remaining time.Duration
	Running   bool
	Finished  bool
	Alarming  bool // Active alarm state (blinking/ringing)
}

type model struct {
	textInput   textinput.Model
	timers      []*Timer
	nextID      int // Keeping nextID if needed, though GetNewID implies calculation
	blink       bool
	width       int
	height      int
	focusIndex  Focus
	focusState  Focus
	alarmCancel context.CancelFunc // To stop the playing sound
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
		timers:     []*Timer{},
		nextID:     1,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tickCmd(),
		blinkCmd(),
	)
}

type tickMsg time.Time
type blinkMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func blinkCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return blinkMsg(t)
	})
}

func playSound(ctx context.Context) {
	// Try standard sound paths
	soundFiles := []string{
		"/usr/share/sounds/freedesktop/stereo/alarm-clock-elapsed.oga",
		"/usr/share/sounds/freedesktop/stereo/complete.oga",
	}

	for _, sf := range soundFiles {
		if _, err := os.Stat(sf); err == nil {
			// Run with context so we can kill it
			_ = exec.CommandContext(ctx, "paplay", sf).Run()
			return
		}
	}
	// Fallback to bell
	fmt.Print("\a")
}

func (m model) GetNewID() int {
	if len(m.timers) == 0 {
		return 1
	}
	return m.timers[len(m.timers)-1].ID + 1
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		// Dismiss any active alarms on key press and stop sound
		anyAlarming := false
		for _, t := range m.timers {
			if t.Alarming {
				t.Alarming = false
				anyAlarming = true
			}
		}

		if m.alarmCancel != nil {
			m.alarmCancel() // Kill the sound process
			m.alarmCancel = nil
		}

		if anyAlarming {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab", "shift+tab", "left", "right", "up", "down":
			s := msg.String()

			switch s {
			case "tab":
				m.focusIndex++
				if m.focusIndex > QUIT {
					m.focusIndex = INPUT
				}

			case "shift+tab":
				m.focusIndex--
				if m.focusIndex < INPUT {
					m.focusIndex = QUIT
				}

			case "left":
				if m.focusIndex == INPUT {
					break
				}
				if m.focusIndex == ADD {
					m.focusIndex = QUIT
					m.focusState = QUIT
					break
				}
				m.focusIndex--
				m.focusState = m.focusIndex

			case "right":
				if m.focusIndex == INPUT {
					break
				}
				if m.focusIndex == QUIT {
					m.focusIndex = ADD
					m.focusState = ADD
					break
				}
				m.focusIndex++
				m.focusState = m.focusIndex

			case "up":
				if m.focusIndex > INPUT {
					m.focusIndex = INPUT
				}

			case "down":
				if m.focusIndex == INPUT {
					if m.focusState > INPUT {
						m.focusIndex = m.focusState
					} else {
						m.focusIndex = ADD
					}
				}
			}

			if m.focusIndex > QUIT {
				m.focusIndex = INPUT
			} else if m.focusIndex < INPUT {
				m.focusIndex = QUIT
			}

			if m.focusIndex == INPUT {
				cmd = m.textInput.Focus()
			} else {
				m.textInput.Blur()
			}
			return m, cmd

		case "enter":
			if m.focusIndex == INPUT {
				parsed, err := time.ParseDuration(m.textInput.Value())
				if err == nil && parsed > 0 {
					newTimer := &Timer{
						ID:        m.GetNewID(),
						Duration:  parsed,
						Remaining: parsed,
						Running:   true,
						Finished:  false,
						Alarming:  false,
					}
					m.timers = append(m.timers, newTimer)
					m.textInput.SetValue("")
				}
			} else if m.focusIndex == ADD {
				parsed, err := time.ParseDuration(m.textInput.Value())
				if err == nil && parsed > 0 {
					newTimer := &Timer{
						ID:        m.GetNewID(),
						Duration:  parsed,
						Remaining: parsed,
						Running:   true,
						Finished:  false,
						Alarming:  false,
					}
					m.timers = append(m.timers, newTimer)
					m.textInput.SetValue("")
				}
			} else if m.focusIndex == START {
				// Global Resume
				for _, t := range m.timers {
					if !t.Finished {
						t.Running = true
					}
				}
			} else if m.focusIndex == STOP {
				// Global Pause
				for _, t := range m.timers {
					t.Running = false
				}
			} else if m.focusIndex == RESET {
				if m.alarmCancel != nil {
					m.alarmCancel()
					m.alarmCancel = nil
				}
				m.timers = []*Timer{}
			} else if m.focusIndex == QUIT {
				if m.alarmCancel != nil {
					m.alarmCancel()
				}
				return m, tea.Quit
			}
		}

	case tickMsg:
		anyFinishedNow := false
		for _, t := range m.timers {
			if t.Running && t.Remaining > 0 {
				t.Remaining -= time.Second
				if t.Remaining <= 0 {
					t.Running = false
					t.Remaining = 0
					t.Finished = true
					t.Alarming = true
					anyFinishedNow = true
				}
			}
		}
		if anyFinishedNow {
			if m.alarmCancel != nil {
				m.alarmCancel()
			}
			ctx, cancel := context.WithCancel(context.Background())
			m.alarmCancel = cancel
			return m, tea.Batch(
				func() tea.Msg { playSound(ctx); return nil },
				tickCmd(),
			)
		}
		return m, tickCmd()

	case blinkMsg:
		m.blink = !m.blink
		return m, blinkCmd()
	}

	if m.focusIndex == INPUT {
		m.textInput, cmd = m.textInput.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	var s strings.Builder

	// Input
	s.WriteString("New Timer: ")
	s.WriteString(m.textInput.View())
	s.WriteString("\n\n")

	// Timer List
	if len(m.timers) == 0 {
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("No timers running"))
		s.WriteString("\n\n")
	} else {
		for _, t := range m.timers {
			s.WriteString(fmt.Sprintf("#%d: ", t.ID))
			if t.Finished {
				msg := "Time's Up!"
				if t.Alarming && m.blink {
					s.WriteString(alarmStyle.Render(msg))
				} else {
					s.WriteString(msg)
				}
			} else {
				status := ""
				if !t.Running {
					status = " (Paused)"
				}
				s.WriteString(fmt.Sprintf("%s remaining%s", t.Remaining.Round(time.Second), status))
			}
			s.WriteString("\n")
		}
		s.WriteString("\n")
	}

	// Buttons
	addButton := fmt.Sprintf("[ %s ]", "Add")
	if m.focusIndex == ADD {
		addButton = fmt.Sprintf(focusedButton, "Add")
	} else {
		addButton = fmt.Sprintf(blurredButton, "Add")
	}

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

	s.WriteString(fmt.Sprintf("%s  %s  %s  %s  %s\n\n", addButton, startButton, stopButton, resetButton, quitButton))

	s.WriteString(helpStyle.Render("(Tab to navigate, Enter to select)"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, s.String())
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
