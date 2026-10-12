# TUI Timer

A simple, terminal-based countdown timer built with Go and Bubble Tea.

![alt text](/pic/demo_pic.png)

## Features

- Flexible duration input:
  - Standard units: `10s`, `5m`, `1h30m`
  - Spaced durations: `1h 30m`, `5 m`, `10 s`
  - Clock notation: `1:30` (1m30s), `10:00` (10m), `1:30:00` (1h30m)
  - Natural words: `5 mins`, `10 seconds`, `1 hour`
  - Plain numbers: `45` (defaults to seconds)
- Visual countdown with smooth second-level precision
- Audible and visual alarm when time expires
- Responsive interface that centers in the terminal window
- Keyboard navigation with full cursor editing support

## Controls

- **Left / Right**: Move cursor inside the input field, or navigate between buttons when buttons are focused
- **Up / Down**: Navigate between input field and controls row
- **Tab / Shift+Tab**: Cycle forward / backward through all controls
- **Enter**: Add timer (when in input) or activate focused button
- **Space**: Activate focused button (when buttons are focused)
- **Esc**: Unfocus input / return to input from buttons
- **q**: Quit the application (when buttons are focused)
- **Ctrl+C**: Quit the application immediately
- **Any Key**: Stop the alarm when a timer finishes

## Installation

Ensure you have Go installed on your system.

```bash
git clone https://github.com/dancoopper/TUI-Timer.git
cd TUI-Timer
go run main.go
```

## Sound Requirements

The timer attempts to play standard system sounds using `paplay` (PulseAudio). If the specific sound files are not found, it falls back to the terminal bell.
