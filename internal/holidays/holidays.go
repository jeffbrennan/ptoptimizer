package holidays

import (
	"time"

	"github.com/jeffb/ptoptimizer/internal/config"
)

// USFederalHolidays returns the US federal holidays for a given year.
// Uses rules (e.g., "3rd Monday of January") rather than hardcoded dates.
func USFederalHolidays(year int) []config.Holiday {
	return []config.Holiday{
		{Name: "New Year's Day", Date: observedDate(time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local))},
		{Name: "Martin Luther King Jr. Day", Date: nthWeekday(year, time.January, time.Monday, 3)},
		{Name: "Presidents' Day", Date: nthWeekday(year, time.February, time.Monday, 3)},
		{Name: "Memorial Day", Date: lastWeekday(year, time.May, time.Monday)},
		{Name: "Juneteenth", Date: observedDate(time.Date(year, time.June, 19, 0, 0, 0, 0, time.Local))},
		{Name: "Independence Day", Date: observedDate(time.Date(year, time.July, 4, 0, 0, 0, 0, time.Local))},
		{Name: "Labor Day", Date: nthWeekday(year, time.September, time.Monday, 1)},
		{Name: "Columbus Day", Date: nthWeekday(year, time.October, time.Monday, 2)},
		{Name: "Veterans Day", Date: observedDate(time.Date(year, time.November, 11, 0, 0, 0, 0, time.Local))},
		{Name: "Thanksgiving Day", Date: nthWeekday(year, time.November, time.Thursday, 4)},
		{Name: "Christmas Day", Date: observedDate(time.Date(year, time.December, 25, 0, 0, 0, 0, time.Local))},
	}
}

// nthWeekday returns the date string for the nth occurrence of a weekday in a month.
func nthWeekday(year int, month time.Month, weekday time.Weekday, n int) string {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	offset := (int(weekday) - int(first.Weekday()) + 7) % 7
	day := 1 + offset + (n-1)*7
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local).Format("2006-01-02")
}

// lastWeekday returns the date string for the last occurrence of a weekday in a month.
func lastWeekday(year int, month time.Month, weekday time.Weekday) string {
	// Start from the last day of the month and work backward
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local)
	for last.Weekday() != weekday {
		last = last.AddDate(0, 0, -1)
	}
	return last.Format("2006-01-02")
}

// observedDate adjusts a holiday date if it falls on a weekend.
// Saturday holidays are observed on Friday; Sunday holidays on Monday.
func observedDate(d time.Time) string {
	switch d.Weekday() {
	case time.Saturday:
		return d.AddDate(0, 0, -1).Format("2006-01-02")
	case time.Sunday:
		return d.AddDate(0, 0, 1).Format("2006-01-02")
	default:
		return d.Format("2006-01-02")
	}
}
