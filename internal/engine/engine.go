package engine

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/jeffb/ptoptimizer/internal/config"
)

const hoursPerDay = 8.0

// ProjectBalance calculates the projected PTO balance on a given target date.
func ProjectBalance(cfg *config.Config, targetDate time.Time) float64 {
	accrual := cfg.Accrual

	asOf, err := time.Parse("2006-01-02", accrual.BalanceAsOfDate)
	if err != nil {
		return accrual.CurrentBalanceHours
	}

	balance := accrual.CurrentBalanceHours

	if accrual.PayPeriodDays <= 0 || accrual.AccrualRateHours <= 0 {
		return balance
	}

	// Calculate accruals
	daysBetween := targetDate.Sub(asOf).Hours() / 24
	periods := math.Floor(daysBetween / float64(accrual.PayPeriodDays))
	if periods > 0 {
		balance += periods * accrual.AccrualRateHours
	}

	// Subtract planned PTO that falls between asOf and targetDate
	for _, pto := range cfg.PlannedTimeOff {
		start, err1 := time.Parse("2006-01-02", pto.StartDate)
		end, err2 := time.Parse("2006-01-02", pto.EndDate)
		if err1 != nil || err2 != nil {
			continue
		}

		// Only count PTO that's after the balance-as-of date and before/on target
		if end.Before(asOf) || start.After(targetDate) {
			continue
		}

		if pto.Hours > 0 {
			balance -= pto.Hours
		} else {
			// Calculate weekdays between start and end
			weekdays := countWeekdays(start, end)
			balance -= float64(weekdays) * hoursPerDay
		}
	}

	// Apply cap
	if accrual.MaxBalanceHours > 0 && balance > accrual.MaxBalanceHours {
		balance = accrual.MaxBalanceHours
	}

	return balance
}

func countWeekdays(start, end time.Time) int {
	count := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			count++
		}
	}
	return count
}

// Suggestion represents a suggested PTO day with its reasoning.
type Suggestion struct {
	Date        time.Time
	Explanation string
	StreakDays   int
}

// SuggestDays returns the top N best PTO days to take in the given year.
func SuggestDays(cfg *config.Config, year int, n int) []Suggestion {
	holidaySet := make(map[string]bool)
	for _, h := range cfg.Holidays {
		holidaySet[h.Date] = true
	}

	ptoSet := make(map[string]bool)
	for _, p := range cfg.PlannedTimeOff {
		start, err1 := time.Parse("2006-01-02", p.StartDate)
		end, err2 := time.Parse("2006-01-02", p.EndDate)
		if err1 != nil || err2 != nil {
			continue
		}
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			ptoSet[d.Format("2006-01-02")] = true
		}
	}

	suggestedSet := make(map[string]bool)
	var suggestions []Suggestion

	for i := 0; i < n; i++ {
		var bestDate time.Time
		var bestScore float64
		var bestStreak int
		var bestExplanation string

		startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
		endOfYear := time.Date(year, 12, 31, 0, 0, 0, 0, time.Local)

		for d := startOfYear; !d.After(endOfYear); d = d.AddDate(0, 0, 1) {
			ds := d.Format("2006-01-02")

			// Skip weekends, holidays, existing PTO, and already-suggested days
			if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
				continue
			}
			if holidaySet[ds] || ptoSet[ds] || suggestedSet[ds] {
				continue
			}

			streak := computeStreak(d, holidaySet, ptoSet, suggestedSet)
			// Score: consecutive days off per PTO day spent
			// The streak includes the candidate day itself as a day off
			score := float64(streak)

			if score > bestScore {
				bestScore = score
				bestDate = d
				bestStreak = streak
				bestExplanation = explainSuggestion(d, streak, holidaySet)
			}
		}

		if bestScore <= 0 {
			break
		}

		suggestedSet[bestDate.Format("2006-01-02")] = true
		suggestions = append(suggestions, Suggestion{
			Date:        bestDate,
			Explanation: bestExplanation,
			StreakDays:   bestStreak,
		})
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Date.Before(suggestions[j].Date)
	})

	return suggestions
}

// computeStreak calculates the total consecutive days off if a candidate day is taken off.
func computeStreak(candidate time.Time, holidays, pto, suggested map[string]bool) int {
	count := 1 // the candidate day itself

	// Expand backward
	for d := candidate.AddDate(0, 0, -1); ; d = d.AddDate(0, 0, -1) {
		if isOff(d, holidays, pto, suggested) {
			count++
		} else {
			break
		}
	}

	// Expand forward
	for d := candidate.AddDate(0, 0, 1); ; d = d.AddDate(0, 0, 1) {
		if isOff(d, holidays, pto, suggested) {
			count++
		} else {
			break
		}
	}

	return count
}

func isOff(d time.Time, holidays, pto, suggested map[string]bool) bool {
	ds := d.Format("2006-01-02")
	if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		return true
	}
	return holidays[ds] || pto[ds] || suggested[ds]
}

func explainSuggestion(d time.Time, streak int, holidays map[string]bool) string {
	dayName := d.Format("Mon Jan 2")

	// Find nearby holidays to mention
	for offset := -5; offset <= 5; offset++ {
		nearby := d.AddDate(0, 0, offset)
		ns := nearby.Format("2006-01-02")
		if holidays[ns] && offset != 0 {
			return fmt.Sprintf("Take %s → %d-day streak (near holiday on %s)",
				dayName, streak, nearby.Format("Jan 2"))
		}
	}

	if streak >= 4 {
		return fmt.Sprintf("Take %s → %d-day streak (extended weekend)", dayName, streak)
	}
	return fmt.Sprintf("Take %s → %d-day streak", dayName, streak)
}

// GetSuggestedDates returns a set of suggested date strings for calendar rendering.
func GetSuggestedDates(suggestions []Suggestion) map[string]bool {
	m := make(map[string]bool)
	for _, s := range suggestions {
		m[s.Date.Format("2006-01-02")] = true
	}
	return m
}
