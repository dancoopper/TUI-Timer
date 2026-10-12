package main

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		// Standard Go duration formats
		{"10s", 10 * time.Second, false},
		{"5m", 5 * time.Minute, false},
		{"1h", 1 * time.Hour, false},
		{"1h30m", 90 * time.Minute, false},
		{"1.5h", 90 * time.Minute, false},

		// Spaces and whitespace
		{" 10s ", 10 * time.Second, false},
		{"1h 30m", 90 * time.Minute, false},
		{"5 m", 5 * time.Minute, false},
		{"10 s", 10 * time.Second, false},

		// Natural language units
		{"10sec", 10 * time.Second, false},
		{"10secs", 10 * time.Second, false},
		{"10 second", 10 * time.Second, false},
		{"10 seconds", 10 * time.Second, false},
		{"5min", 5 * time.Minute, false},
		{"5mins", 5 * time.Minute, false},
		{"5 minute", 5 * time.Minute, false},
		{"5 minutes", 5 * time.Minute, false},
		{"1hr", 1 * time.Hour, false},
		{"1hrs", 1 * time.Hour, false},
		{"1 hour", 1 * time.Hour, false},
		{"1 hours", 1 * time.Hour, false},

		// Plain numbers defaulting to seconds
		{"45", 45 * time.Second, false},
		{"60", 60 * time.Second, false},

		// Digital clock format
		{"1:30", 90 * time.Second, false},
		{"05:00", 5 * time.Minute, false},
		{"0:45", 45 * time.Second, false},
		{"1:30:00", 90 * time.Minute, false},

		// Shorthand hour without m
		{"1h30", 90 * time.Minute, false},
		{"1h 30", 90 * time.Minute, false},

		// Invalid cases
		{"", 0, true},
		{"   ", 0, true},
		{"0", 0, true},
		{"0s", 0, true},
		{"-5s", 0, true},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		got, err := parseDuration(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("parseDuration(%q): expected error, got %v", tt.input, got)
			}
		} else {
			if err != nil {
				t.Errorf("parseDuration(%q): unexpected error: %v", tt.input, err)
			} else if got != tt.expected {
				t.Errorf("parseDuration(%q): got %v, expected %v", tt.input, got, tt.expected)
			}
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{0, "0s"},
		{10 * time.Second, "10s"},
		{90 * time.Second, "1m 30s"},
		{65 * time.Second, "1m 05s"},
		{1 * time.Hour, "1h 00m 00s"},
		{90 * time.Minute, "1h 30m 00s"},
		{90*time.Minute + 5*time.Second, "1h 30m 05s"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.input)
		if got != tt.expected {
			t.Errorf("formatDuration(%v): got %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestInputLeftRightArrowKeys(t *testing.T) {
	m := initialModel(0)
	// Type "10s"
	m.textInput.SetValue("10s")
	m.textInput.SetCursor(3) // Cursor at end

	// Press Left arrow
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	modelAfter := newM.(model)

	if modelAfter.focusIndex != INPUT {
		t.Fatalf("expected focus to remain on INPUT, got %v", modelAfter.focusIndex)
	}
	if modelAfter.textInput.Position() != 2 {
		t.Errorf("expected cursor position 2 after KeyLeft, got %d", modelAfter.textInput.Position())
	}

	// Press Right arrow
	newM2, _ := modelAfter.Update(tea.KeyMsg{Type: tea.KeyRight})
	modelAfter2 := newM2.(model)

	if modelAfter2.textInput.Position() != 3 {
		t.Errorf("expected cursor position 3 after KeyRight, got %d", modelAfter2.textInput.Position())
	}
}

func TestInputEnterAddsTimer(t *testing.T) {
	m := initialModel(0)
	m.textInput.SetValue("5m")

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	modelAfter := newM.(model)

	if len(modelAfter.timers) != 1 {
		t.Fatalf("expected 1 timer, got %d", len(modelAfter.timers))
	}
	if modelAfter.timers[0].Duration != 5*time.Minute {
		t.Errorf("expected 5m duration, got %v", modelAfter.timers[0].Duration)
	}
	if modelAfter.textInput.Value() != "" {
		t.Errorf("expected textInput to be cleared, got %q", modelAfter.textInput.Value())
	}
	if modelAfter.errMessage != "" {
		t.Errorf("expected errMessage to be empty, got %q", modelAfter.errMessage)
	}
}

func TestInputInvalidShowsError(t *testing.T) {
	m := initialModel(0)
	m.textInput.SetValue("invalid")

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	modelAfter := newM.(model)

	if len(modelAfter.timers) != 0 {
		t.Errorf("expected 0 timers, got %d", len(modelAfter.timers))
	}
	if modelAfter.errMessage == "" {
		t.Errorf("expected error message to be set for invalid duration")
	}
	if modelAfter.textInput.Value() != "invalid" {
		t.Errorf("expected textInput to preserve invalid input for user editing, got %q", modelAfter.textInput.Value())
	}
}

func TestNavigation(t *testing.T) {
	m := initialModel(0)

	// Down arrow from INPUT moves to ADD button
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	mod1 := m1.(model)
	if mod1.focusIndex != ADD {
		t.Errorf("expected focus on ADD after KeyDown, got %v", mod1.focusIndex)
	}

	// Right arrow moves from ADD to START
	m2, _ := mod1.Update(tea.KeyMsg{Type: tea.KeyRight})
	mod2 := m2.(model)
	if mod2.focusIndex != START {
		t.Errorf("expected focus on START after KeyRight, got %v", mod2.focusIndex)
	}

	// Left arrow wraps from ADD to QUIT
	m3, _ := mod1.Update(tea.KeyMsg{Type: tea.KeyLeft})
	mod3 := m3.(model)
	if mod3.focusIndex != QUIT {
		t.Errorf("expected focus on QUIT after KeyLeft from ADD, got %v", mod3.focusIndex)
	}

	// Up arrow returns to INPUT
	m4, _ := mod3.Update(tea.KeyMsg{Type: tea.KeyUp})
	mod4 := m4.(model)
	if mod4.focusIndex != INPUT {
		t.Errorf("expected focus on INPUT after KeyUp, got %v", mod4.focusIndex)
	}

	// Esc from INPUT moves to buttons
	m5, _ := mod4.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mod5 := m5.(model)
	if mod5.focusIndex == INPUT {
		t.Errorf("expected focus to move to buttons on Esc from INPUT")
	}

	// Esc from buttons returns to INPUT
	m6, _ := mod5.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mod6 := m6.(model)
	if mod6.focusIndex != INPUT {
		t.Errorf("expected focus to return to INPUT on Esc from buttons, got %v", mod6.focusIndex)
	}
}

func TestQDoesNotQuitWhenTyping(t *testing.T) {
	m := initialModel(0)
	// Press 'q' while in INPUT
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		// tea.Quit command would be non-nil if quitting
		// Ensure it didn't quit
	}
	mod := newM.(model)
	if mod.textInput.Value() != "q" {
		t.Errorf("expected 'q' to be typed into textInput, got %q", mod.textInput.Value())
	}
}

func TestButtonSpaceActivation(t *testing.T) {
	m := initialModel(10 * time.Second)
	// Move focus to STOP
	m.focusIndex = STOP
	if !m.timers[0].Running {
		t.Fatal("timer should be initially running")
	}

	// Press Space on STOP button
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	mod := newM.(model)

	if mod.timers[0].Running {
		t.Errorf("expected timer to be stopped after Space on STOP button")
	}
}

func TestSmoothTicking(t *testing.T) {
	m := initialModel(10 * time.Second)
	start := time.Now()
	m.lastTick = start

	// Simulate 200ms tick
	tickTime := start.Add(200 * time.Millisecond)
	newM, _ := m.Update(tickMsg(tickTime))
	mod := newM.(model)

	expectedRemaining := 10*time.Second - 200*time.Millisecond
	if mod.timers[0].Remaining != expectedRemaining {
		t.Errorf("expected remaining %v, got %v", expectedRemaining, mod.timers[0].Remaining)
	}

	// Formatted display should still show "10s" because 9.8s rounds up to 10s ceiling
	formatted := formatDuration(mod.timers[0].Remaining)
	if formatted != "10s" {
		t.Errorf("expected formatted remaining '10s' after 200ms, got %q", formatted)
	}
}
