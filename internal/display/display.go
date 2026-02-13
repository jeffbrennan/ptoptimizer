package display

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	holidayStyle  = lipgloss.NewStyle().Background(lipgloss.Color("196")).Foreground(lipgloss.Color("231")).Bold(true)
	ptoStyle      = lipgloss.NewStyle().Background(lipgloss.Color("34")).Foreground(lipgloss.Color("231")).Bold(true)
	suggestedStyle = lipgloss.NewStyle().Background(lipgloss.Color("220")).Foreground(lipgloss.Color("16")).Bold(true)
	weekendStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	normalStyle   = lipgloss.NewStyle()
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))

	barFilled = lipgloss.NewStyle().Foreground(lipgloss.Color("34"))
	barEmpty  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// CalendarData holds sets of special dates for rendering.
type CalendarData struct {
	Holidays  map[string]bool
	PTO       map[string]bool
	Suggested map[string]bool
}

// RenderBalanceBar renders a progress bar showing PTO balance.
func RenderBalanceBar(current, max float64) string {
	if max <= 0 {
		return titleStyle.Render(fmt.Sprintf("  PTO Balance: %.0f hrs", current))
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
	return titleStyle.Render(fmt.Sprintf("  PTO Balance: %s %.0f/%.0f hrs (%.0f days remaining)",
		bar, current, max, daysRemaining))
}

// RenderYearCalendar renders a full year calendar grid with color-coded days.
func RenderYearCalendar(year int, data CalendarData) string {
	var sb strings.Builder

	// Render months in pairs (2 columns)
	for month := time.January; month <= time.December; month += 2 {
		left := renderMonth(year, month, data)
		var right []string
		if month+1 <= time.December {
			right = strings.Split(renderMonth(year, month+1, data), "\n")
		}
		leftLines := strings.Split(left, "\n")

		// Pad to same height
		maxLines := len(leftLines)
		if len(right) > maxLines {
			maxLines = len(right)
		}
		for len(leftLines) < maxLines {
			leftLines = append(leftLines, "")
		}
		for len(right) < maxLines {
			right = append(right, "")
		}

		for i := 0; i < maxLines; i++ {
			l := padRight(leftLines[i], 32)
			r := ""
			if i < len(right) {
				r = right[i]
			}
			sb.WriteString("  " + l + "    " + r + "\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func renderMonth(year int, month time.Month, data CalendarData) string {
	var sb strings.Builder

	title := fmt.Sprintf("── %s %d ", month.String(), year)
	title += strings.Repeat("─", 26-visibleLen(title))
	sb.WriteString(headerStyle.Render(title) + "\n")

	sb.WriteString(weekendStyle.Render("Mo Tu We Th Fr") + "  " + weekendStyle.Render("Sa Su") + "\n")

	first := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local)

	// Offset for the first day (Monday = 0)
	offset := (int(first.Weekday()) + 6) % 7
	sb.WriteString(strings.Repeat("   ", offset))

	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		ds := d.Format("2006-01-02")
		dayStr := fmt.Sprintf("%2d", d.Day())
		dow := (int(d.Weekday()) + 6) % 7 // Monday = 0

		// Insert separator before weekend columns
		if dow == 5 && d.Day() > 1 {
			// Already have space from previous day
		}

		switch {
		case data.Holidays[ds]:
			dayStr = holidayStyle.Render(dayStr)
		case data.PTO[ds]:
			dayStr = ptoStyle.Render(dayStr)
		case data.Suggested[ds]:
			dayStr = suggestedStyle.Render(dayStr)
		case d.Weekday() == time.Saturday || d.Weekday() == time.Sunday:
			dayStr = weekendStyle.Render(dayStr)
		default:
			dayStr = normalStyle.Render(dayStr)
		}

		if dow == 5 {
			sb.WriteString("  " + dayStr)
		} else {
			sb.WriteString(dayStr)
		}

		if dow == 6 {
			sb.WriteString("\n")
		} else {
			sb.WriteString(" ")
		}
	}

	// Final newline if month doesn't end on Sunday
	if last.Weekday() != time.Sunday {
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderLegend renders the color legend.
func RenderLegend() string {
	return fmt.Sprintf("  Legend: %s Holiday  %s Planned PTO  %s Suggested  %s Weekend",
		holidayStyle.Render("██"),
		ptoStyle.Render("██"),
		suggestedStyle.Render("██"),
		weekendStyle.Render("░░"),
	)
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
