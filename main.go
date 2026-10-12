package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
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
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red

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
	nextID      int
	blink       bool
	width       int
	height      int
	focusIndex  Focus
	focusState  Focus
	alarmCancel context.CancelFunc // To stop the playing sound
	errMessage  string             // User-facing validation error message
	lastTick    time.Time          // To measure exact elapsed time without tick drift
}

func initialModel(initialDuration time.Duration) model {
	ti := textinput.New()
	ti.Placeholder = "10s, 5m, 1h30m, 1:30"
	ti.Prompt = ""
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 32
	ti.Cursor.Style = cursorStyle

	timers := []*Timer{}
	nextID := 1
	if initialDuration > 0 {
		timers = append(timers, &Timer{
			ID:        1,
			Duration:  initialDuration,
			Remaining: initialDuration,
			Running:   true,
			Finished:  false,
			Alarming:  false,
		})
		nextID = 2
	}

	return model{
		textInput:  ti,
		focusIndex: INPUT,
		timers:     timers,
		nextID:     nextID,
		lastTick:   time.Now(),
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
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
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
			if err := exec.CommandContext(ctx, "paplay", sf).Run(); err == nil {
				return
			}
		}
	}
	// Fallback to bell
	fmt.Print("\a")
}

func (m *model) GetNewID() int {
	id := m.nextID
	m.nextID++
	return id
}

func parseDuration(input string) (time.Duration, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return 0, fmt.Errorf("please enter a duration")
	}

	// Digital clock format MM:SS or HH:MM:SS
	if strings.Contains(s, ":") {
		parts := strings.Split(s, ":")
		if len(parts) == 2 {
			m, errM := strconv.Atoi(strings.TrimSpace(parts[0]))
			sec, errS := strconv.Atoi(strings.TrimSpace(parts[1]))
			if errM == nil && errS == nil && m >= 0 && sec >= 0 && sec < 60 {
				d := time.Duration(m)*time.Minute + time.Duration(sec)*time.Second
				if d <= 0 {
					return 0, fmt.Errorf("duration must be greater than zero")
				}
				return d, nil
			}
		} else if len(parts) == 3 {
			h, errH := strconv.Atoi(strings.TrimSpace(parts[0]))
			m, errM := strconv.Atoi(strings.TrimSpace(parts[1]))
			sec, errS := strconv.Atoi(strings.TrimSpace(parts[2]))
			if errH == nil && errM == nil && errS == nil && h >= 0 && m >= 0 && m < 60 && sec >= 0 && sec < 60 {
				d := time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second
				if d <= 0 {
					return 0, fmt.Errorf("duration must be greater than zero")
				}
				return d, nil
			}
		}
	}

	// Plain number: default to seconds
	if n, err := strconv.Atoi(s); err == nil {
		if n <= 0 {
			return 0, fmt.Errorf("duration must be greater than zero")
		}
		return time.Duration(n) * time.Second, nil
	}

	// Standard Go duration
	if d, err := time.ParseDuration(s); err == nil {
		if d <= 0 {
			return 0, fmt.Errorf("duration must be greater than zero")
		}
		return d, nil
	}

	// Normalize natural language
	normalized := strings.ToLower(s)
	replacements := []struct {
		old string
		new string
	}{
		{"milliseconds", "ms"},
		{"millisecond", "ms"},
		{"seconds", "s"},
		{"second", "s"},
		{"secs", "s"},
		{"sec", "s"},
		{"minutes", "m"},
		{"minute", "m"},
		{"mins", "m"},
		{"min", "m"},
		{"hours", "h"},
		{"hour", "h"},
		{"hrs", "h"},
		{"hr", "h"},
	}

	for _, r := range replacements {
		normalized = strings.ReplaceAll(normalized, r.old, r.new)
	}

	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "\t", "")

	// Handle 1h30 (without m) -> 1h30m
	re := regexp.MustCompile(`^(\d+h)(\d+)$`)
	if re.MatchString(normalized) {
		normalized = re.ReplaceAllString(normalized, "${1}${2}m")
	}

	d, err := time.ParseDuration(normalized)
	if err == nil {
		if d <= 0 {
			return 0, fmt.Errorf("duration must be greater than zero")
		}
		return d, nil
	}

	return 0, fmt.Errorf("invalid duration (e.g. 10s, 5m, 1h30m, 1:30)")
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	totalSec := int(math.Ceil(d.Seconds()))
	h := totalSec / 3600
	m := (totalSec % 3600) / 60
	s := totalSec % 60

	if h > 0 {
		return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func renderButton(text string, focused bool) string {
	btn := fmt.Sprintf("[ %s ]", text)
	if focused {
		return focusedStyle.Render(btn)
	}
	return blurredStyle.Render(btn)
}

func (m *model) addTimer() bool {
	val := m.textInput.Value()
	if strings.TrimSpace(val) == "" {
		return false
	}
	parsed, err := parseDuration(val)
	if err != nil {
		m.errMessage = err.Error()
		return false
	}

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
	m.errMessage = ""
	m.lastTick = time.Now()
	return true
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Ctrl+C always quits immediately
		if msg.String() == "ctrl+c" {
			if m.alarmCancel != nil {
				m.alarmCancel()
			}
			return m, tea.Quit
		}

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

		// When focused on INPUT
		if m.focusIndex == INPUT {
			switch msg.String() {
			case "enter":
				m.addTimer()
				return m, nil

			case "esc":
				m.errMessage = ""
				m.textInput.Blur()
				if m.focusState > INPUT {
					m.focusIndex = m.focusState
				} else {
					m.focusIndex = ADD
				}
				return m, nil

			case "tab":
				m.errMessage = ""
				m.textInput.Blur()
				m.focusIndex = ADD
				m.focusState = ADD
				return m, nil

			case "shift+tab":
				m.errMessage = ""
				m.textInput.Blur()
				m.focusIndex = QUIT
				m.focusState = QUIT
				return m, nil

			case "down":
				m.errMessage = ""
				m.textInput.Blur()
				if m.focusState > INPUT {
					m.focusIndex = m.focusState
				} else {
					m.focusIndex = ADD
				}
				return m, nil

			default:
				// Any typing or cursor movement (Left, Right, Home, End, Backspace, etc.)
				m.errMessage = ""
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}

		// When focused on BUTTONS (m.focusIndex != INPUT)
		switch msg.String() {
		case "q":
			if m.alarmCancel != nil {
				m.alarmCancel()
			}
			return m, tea.Quit

		case "tab":
			m.focusIndex++
			if m.focusIndex > QUIT {
				m.focusIndex = INPUT
				m.focusState = INPUT
				cmd = m.textInput.Focus()
				return m, cmd
			}
			m.focusState = m.focusIndex
			return m, nil

		case "shift+tab":
			m.focusIndex--
			if m.focusIndex < INPUT {
				m.focusIndex = QUIT
			}
			if m.focusIndex == INPUT {
				m.focusState = INPUT
				cmd = m.textInput.Focus()
				return m, cmd
			}
			m.focusState = m.focusIndex
			return m, nil

		case "left":
			if m.focusIndex == ADD {
				m.focusIndex = QUIT
			} else {
				m.focusIndex--
			}
			m.focusState = m.focusIndex
			return m, nil

		case "right":
			if m.focusIndex == QUIT {
				m.focusIndex = ADD
			} else {
				m.focusIndex++
			}
			m.focusState = m.focusIndex
			return m, nil

		case "up", "esc":
			m.focusIndex = INPUT
			m.focusState = INPUT
			cmd = m.textInput.Focus()
			return m, cmd

		case "enter", " ":
			switch m.focusIndex {
			case ADD:
				if strings.TrimSpace(m.textInput.Value()) != "" {
					m.addTimer()
				}
				m.focusIndex = INPUT
				m.focusState = INPUT
				cmd = m.textInput.Focus()
				return m, cmd

			case START:
				m.lastTick = time.Now()
				for _, t := range m.timers {
					if !t.Finished {
						t.Running = true
					}
				}
				return m, nil

			case STOP:
				for _, t := range m.timers {
					t.Running = false
				}
				return m, nil

			case RESET:
				if m.alarmCancel != nil {
					m.alarmCancel()
					m.alarmCancel = nil
				}
				m.timers = []*Timer{}
				m.nextID = 1
				m.errMessage = ""
				return m, nil

			case QUIT:
				if m.alarmCancel != nil {
					m.alarmCancel()
				}
				return m, tea.Quit
			}
		}

	case tickMsg:
		now := time.Time(msg)
		if m.lastTick.IsZero() {
			m.lastTick = now
		}
		elapsed := now.Sub(m.lastTick)
		m.lastTick = now

		anyFinishedNow := false
		for _, t := range m.timers {
			if t.Running && t.Remaining > 0 {
				t.Remaining -= elapsed
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

	// Input label & field
	label := "New Timer: "
	if m.focusIndex == INPUT {
		s.WriteString(focusedStyle.Render(label))
	} else {
		s.WriteString(blurredStyle.Render(label))
	}
	s.WriteString(m.textInput.View())
	s.WriteString("\n")

	// Error message line
	if m.errMessage != "" {
		s.WriteString(errorStyle.Render(m.errMessage))
		s.WriteString("\n\n")
	} else {
		s.WriteString("\n")
	}

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
				s.WriteString(fmt.Sprintf("%s remaining%s", formatDuration(t.Remaining), status))
			}
			s.WriteString("\n")
		}
		s.WriteString("\n")
	}

	// Buttons
	addButton := renderButton("Add", m.focusIndex == ADD)
	startButton := renderButton("Start", m.focusIndex == START)
	stopButton := renderButton("Stop", m.focusIndex == STOP)
	resetButton := renderButton("Reset", m.focusIndex == RESET)
	quitButton := renderButton("Quit", m.focusIndex == QUIT)

	s.WriteString(fmt.Sprintf("%s  %s  %s  %s  %s\n\n", addButton, startButton, stopButton, resetButton, quitButton))

	// Help hints
	if m.focusIndex == INPUT {
		s.WriteString(helpStyle.Render("(Enter: add timer • Tab/Down: controls • Esc: unfocus)"))
	} else {
		s.WriteString(helpStyle.Render("(Tab/Arrows: navigate • Enter/Space: select • Up/Esc: input • q: quit)"))
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, s.String())
}

func main() {
	var duration time.Duration
	if len(os.Args) > 1 {
		var err error
		duration, err = parseDuration(os.Args[1])
		if err != nil {
			fmt.Printf("Invalid duration: %v\n", err)
			os.Exit(1)
		}
	}
	p := tea.NewProgram(initialModel(duration), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
