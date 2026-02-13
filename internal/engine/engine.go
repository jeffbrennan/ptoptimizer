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

const maxStreakDays = 14

// SuggestDays returns the top N best PTO days to take in the given year.
// It uses a two-phase approach:
//  1. Pick week-long (5-weekday) vacation blocks for true vacations
//  2. Fill remaining days individually for long weekends
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
	remaining := n

	today := time.Now()
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	if startDate.Before(today) {
		startDate = today
	}
	endOfYear := time.Date(year, 12, 31, 0, 0, 0, 0, time.Local)

	// Phase 1: Pick vacation blocks (5-10 weekdays each) while we have enough days
	for remaining >= 5 {
		bestBlock, bestScore := findBestBlock(startDate, endOfYear, remaining, holidaySet, ptoSet, suggestedSet)
		if bestScore <= 0 || len(bestBlock) == 0 {
			break
		}

		// Add all days in the block
		for _, d := range bestBlock {
			ds := d.Format("2006-01-02")
			suggestedSet[ds] = true
			streak := computeStreak(d, holidaySet, ptoSet, suggestedSet)
			if streak > maxStreakDays {
				streak = maxStreakDays
			}
			suggestions = append(suggestions, Suggestion{
				Date:        d,
				Explanation: explainSuggestion(d, streak, holidaySet),
				StreakDays:   streak,
			})
		}
		remaining -= len(bestBlock)
	}

	// Phase 2: Fill remaining days individually for long weekends
	for remaining > 0 {
		bestDate, bestScore, bestStreak := findBestDay(startDate, endOfYear, holidaySet, ptoSet, suggestedSet)
		if bestScore <= 0 {
			break
		}

		suggestedSet[bestDate.Format("2006-01-02")] = true
		suggestions = append(suggestions, Suggestion{
			Date:        bestDate,
			Explanation: explainSuggestion(bestDate, bestStreak, holidaySet),
			StreakDays:   bestStreak,
		})
		remaining--
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Date.Before(suggestions[j].Date)
	})

	// Recompute streak for each suggestion now that all days are placed
	for i, s := range suggestions {
		streak := computeStreak(s.Date, holidaySet, ptoSet, suggestedSet)
		if streak > maxStreakDays {
			streak = maxStreakDays
		}
		suggestions[i].StreakDays = streak
		suggestions[i].Explanation = explainSuggestion(s.Date, streak, holidaySet)
	}

	return suggestions
}

// findBestBlock finds the best contiguous block of weekdays to take off.
// It tries block sizes from 5 up to maxSize (capped by remaining days and 10).
func findBestBlock(start, end time.Time, remaining int, holidays, pto, suggested map[string]bool) ([]time.Time, float64) {
	maxSize := remaining
	if maxSize > 10 {
		maxSize = 10
	}

	var bestBlock []time.Time
	var bestScore float64

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}

		// Build the longest available weekday run starting at d
		var block []time.Time
		for c := d; !c.After(end) && len(block) < maxSize; c = c.AddDate(0, 0, 1) {
			if c.Weekday() == time.Saturday || c.Weekday() == time.Sunday {
				continue
			}
			cs := c.Format("2006-01-02")
			if holidays[cs] || pto[cs] || suggested[cs] {
				break
			}
			block = append(block, c)
		}

		// Only consider blocks of 5+ weekdays
		if len(block) < 5 {
			continue
		}

		// Try each valid block length (5 to len(block))
		for size := 5; size <= len(block); size++ {
			candidate := block[:size]

			// Temporarily mark these days to compute the full streak
			for _, bd := range candidate {
				suggested[bd.Format("2006-01-02")] = true
			}

			// Streak = total consecutive calendar days off for first day in block
			streak := computeStreak(candidate[0], holidays, pto, suggested)
			if streak > maxStreakDays {
				streak = maxStreakDays
			}

			// Score: total streak, penalized by proximity to existing suggestions
			penalty := blockProximityPenalty(candidate[0], candidate[len(candidate)-1], suggested)
			score := float64(streak) * penalty

			// Unmark
			for _, bd := range candidate {
				delete(suggested, bd.Format("2006-01-02"))
			}

			if score > bestScore {
				bestScore = score
				bestBlock = make([]time.Time, size)
				copy(bestBlock, candidate)
			}
		}
	}

	return bestBlock, bestScore
}

// findBestDay finds the single best day to take off (for filling remaining days).
func findBestDay(start, end time.Time, holidays, pto, suggested map[string]bool) (time.Time, float64, int) {
	var bestDate time.Time
	var bestScore float64
	var bestStreak int

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		ds := d.Format("2006-01-02")
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		if holidays[ds] || pto[ds] || suggested[ds] {
			continue
		}

		streak := computeStreak(d, holidays, pto, suggested)
		if streak > maxStreakDays {
			streak = maxStreakDays
		}

		penalty := blockProximityPenalty(d, d, suggested)
		score := float64(streak) * penalty

		if score > bestScore {
			bestScore = score
			bestDate = d
			bestStreak = streak
		}
	}

	return bestDate, bestScore, bestStreak
}

// blockProximityPenalty penalizes blocks that are close to already-suggested days.
func blockProximityPenalty(blockStart, blockEnd time.Time, suggested map[string]bool) float64 {
	if len(suggested) == 0 {
		return 1.0
	}
	minDist := 366.0
	for ds := range suggested {
		t, err := time.Parse("2006-01-02", ds)
		if err != nil {
			continue
		}
		// Distance from nearest edge of the block
		var dist float64
		if t.Before(blockStart) {
			dist = blockStart.Sub(t).Hours() / 24
		} else if t.After(blockEnd) {
			dist = t.Sub(blockEnd).Hours() / 24
		} else {
			continue // inside the block, skip
		}
		if dist < minDist {
			minDist = dist
		}
	}
	return math.Min(1.0, minDist/60.0)
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
	// Find nearby holidays to mention
	for offset := -7; offset <= 7; offset++ {
		if offset == 0 {
			continue
		}
		nearby := d.AddDate(0, 0, offset)
		ns := nearby.Format("2006-01-02")
		if holidays[ns] {
			return fmt.Sprintf("near %s holiday", nearby.Format("Jan 02"))
		}
	}

	if streak >= 4 {
		return "extended weekend"
	}
	return "long weekend"
}

// GetSuggestedDates returns a set of suggested date strings for calendar rendering.
func GetSuggestedDates(suggestions []Suggestion) map[string]bool {
	m := make(map[string]bool)
	for _, s := range suggestions {
		m[s.Date.Format("2006-01-02")] = true
	}
	return m
}
