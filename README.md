# TUI Timer

A simple, terminal-based countdown timer built with Go and Bubble Tea.

![alt text](/pic/demo_pic.png)

## Features

- specific duration input (e.g., 5m, 1h30m, 10s)
- Visual countdown
- Audible and visual alarm when time expires
- Responsive interface that centers in the terminal window
- Keyboard navigation

## Controls

- **Arrow Keys (Up / Down / Left / Right) or (Tab / Shift+Tab)**: Navigate between controls (Input, Start, Stop, Reset, Quit)
- **(Enter)**: Select focused button
- **(Ctrl+C / q)**: Quit the application
- **(Any Key)**: Stop the alarm when the timer finishes

## Installation

Ensure you have Go installed on your system.

```bash
git clone https://github.com/dancoopper/TUI-Timer.git
cd TUI-Timer
go run main.go
```

## Sound Requirements

The timer attempts to play standard system sounds using `paplay` (PulseAudio). If the specific sound files are not found, it falls back to the terminal bell.
