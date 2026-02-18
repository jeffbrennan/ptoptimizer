package display

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	ptoStyle       = lipgloss.NewStyle().Background(lipgloss.Color("209")).Foreground(lipgloss.Color("16")).Bold(true)
	suggestedColor = lipgloss.NewStyle().Foreground(lipgloss.Color("209")).Bold(true)
	todayStyle     = lipgloss.NewStyle().Background(lipgloss.Color("73")).Foreground(lipgloss.Color("16")).Bold(true)
	blackoutStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	weekendStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	headerStyle    = lipgloss.NewStyle().Bold(true)
	titleStyle     = lipgloss.NewStyle().Bold(true)

	barFilled = lipgloss.NewStyle().Foreground(lipgloss.Color("34"))
	barEmpty  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// CalendarData holds sets of special dates for rendering.
type CalendarData struct {
	Holidays  map[string]bool
	PTO       map[string]bool
	Suggested map[string]bool
	Blackout  map[string]bool
}

// RenderBalanceBar renders a progress bar showing PTO balance.
func RenderBalanceBar(current, max float64) string {
	if max <= 0 {
		return titleStyle.Render(fmt.Sprintf("Unplanned PTO Balance on 12/31: %.0f hrs", current))
	}

	barWidth := 30
	filled := int((current / max) * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}

	bar := barFilled.Render(strings.Repeat("█", filled)) +
		barEmpty.Render(strings.Repeat("░", barWidth-filled))

	daysRemaining := current / 8.0
	return titleStyle.Render(fmt.Sprintf("Unplanned PTO Balance on 12/31: %s %.0f/%.0f hrs (%.0f days remaining)",
		bar, current, max, daysRemaining))
}

// monthWidth is the visible character width of one rendered month column.
const monthWidth = 20

// RenderYearCalendar renders a full year calendar in a 6×2 grid.
func RenderYearCalendar(year int, data CalendarData) string {
	var sb strings.Builder

	for row := 0; row < 3; row++ {
		// Render each month in this row into lines
		monthLines := make([][]string, 4)
		maxHeight := 0
		for col := 0; col < 4; col++ {
			m := time.Month(row*4 + col + 1)
			monthLines[col] = renderMonthLines(year, m, data)
			if len(monthLines[col]) > maxHeight {
				maxHeight = len(monthLines[col])
			}
		}

		// Pad all months to same height
		for col := 0; col < 4; col++ {
			for len(monthLines[col]) < maxHeight {
				monthLines[col] = append(monthLines[col], "")
			}
		}

		// Print side by side
		for line := 0; line < maxHeight; line++ {
			for col := 0; col < 4; col++ {
				if col > 0 {
					sb.WriteString("  ")
				}
				sb.WriteString(padRight(monthLines[col][line], monthWidth))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func renderMonthLines(year int, month time.Month, data CalendarData) []string {
	var lines []string

	// Month header: full name, left aligned
	name := month.String()
	header := name
	lines = append(lines, headerStyle.Render(header))

	first := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local)

	// Offset for the first day (Monday = 0)
	offset := (int(first.Weekday()) + 6) % 7

	var weekLine strings.Builder
	weekLine.WriteString(strings.Repeat("   ", offset))

	today := time.Now().Format("2006-01-02")

	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		dow := (int(d.Weekday()) + 6) % 7
		ds := d.Format("2006-01-02")
		dayStr := fmt.Sprintf("%2d", d.Day())

		switch {
		case data.PTO[ds]:
			dayStr = ptoStyle.Render(dayStr)
		case data.Blackout[ds]:
			dayStr = blackoutStyle.Render(dayStr)
		case data.Suggested[ds]:
			dayStr = suggestedColor.Render(dayStr)
		case ds == today:
			dayStr = todayStyle.Render(dayStr)
		case d.Weekday() == time.Saturday || d.Weekday() == time.Sunday:
			dayStr = weekendStyle.Render(dayStr)
		}

		weekLine.WriteString(dayStr)

		if dow == 6 {
			lines = append(lines, weekLine.String())
			weekLine.Reset()
		} else {
			weekLine.WriteString(" ")
		}
	}

	if weekLine.Len() > 0 {
		lines = append(lines, weekLine.String())
	}

	return lines
}

func padRight(s string, width int) string {
	visible := visibleLen(s)
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

// visibleLen estimates the visible length by stripping ANSI escape sequences.
func visibleLen(s string) int {
	inEscape := false
	count := 0
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		count++
	}
	return count
}
