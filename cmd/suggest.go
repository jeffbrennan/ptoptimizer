package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/jeffb/ptoptimizer/internal/config"
	"github.com/jeffb/ptoptimizer/internal/display"
	"github.com/jeffb/ptoptimizer/internal/engine"
	"github.com/spf13/cobra"
)

var suggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Suggest optimal PTO days",
	RunE: func(cmd *cobra.Command, args []string) error {
		year, _ := cmd.Flags().GetInt("year")

		maxVacations, _ := cmd.Flags().GetInt("vacations")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if len(cfg.Holidays) == 0 {
			fmt.Println("No holidays configured. Run 'ptopt holidays reset' first.")
			return nil
		}

		// Calculate how many days to suggest to reach 0 balance by year end
		balance := engine.ProjectBalance(cfg, time.Date(year, 12, 31, 0, 0, 0, 0, time.Local))
		days := int(balance / 8)
		if days < 0 {
			days = 0
		}

		// Build blackout set
		blackoutSet := make(map[string]bool)
		for _, b := range cfg.BlackoutDates {
			start, err1 := time.Parse("2006-01-02", b.StartDate)
			end, err2 := time.Parse("2006-01-02", b.EndDate)
			if err1 != nil || err2 != nil {
				continue
			}
			for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
				blackoutSet[d.Format("2006-01-02")] = true
			}
		}

		suggestions := engine.SuggestDays(cfg, year, days, maxVacations, blackoutSet)
		fmt.Println()
		fmt.Println(display.RenderBalanceBar(balance, cfg.Accrual.MaxBalanceHours))
		fmt.Println()

		// Build calendar data
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

		suggestedSet := engine.GetSuggestedDates(suggestions)

		data := display.CalendarData{
			Holidays:  holidaySet,
			PTO:       ptoSet,
			Suggested: suggestedSet,
			Blackout:  blackoutSet,
		}

		fmt.Println(display.RenderYearCalendar(year, data))

		// Build planned PTO weekday list with labels
		type plannedDay struct {
			date  time.Time
			label string
		}
		var plannedDates []plannedDay
		for _, p := range cfg.PlannedTimeOff {
			start, err1 := time.Parse("2006-01-02", p.StartDate)
			end, err2 := time.Parse("2006-01-02", p.EndDate)
			if err1 != nil || err2 != nil {
				continue
			}
			for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
				if d.Year() == year && d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
					plannedDates = append(plannedDates, plannedDay{date: d, label: p.Label})
				}
			}
		}

		sort.Slice(plannedDates, func(i, j int) bool {
			return plannedDates[i].date.Before(plannedDates[j].date)
		})

		// Two-column output: Planned | Suggested
		planCol := fmt.Sprintf("Planned (%d days)", len(plannedDates))
		sugCol := fmt.Sprintf("Suggested (%d days)", len(suggestions))
		fmt.Printf("  %-34s  %s\n\n", planCol, sugCol)

		maxRows := len(plannedDates)
		if len(suggestions) > maxRows {
			maxRows = len(suggestions)
		}
		for i := 0; i < maxRows; i++ {
			left := ""
			if i < len(plannedDates) {
				left = plannedDates[i].date.Format("Mon Jan 02")
				if plannedDates[i].label != "" {
					left += "  " + plannedDates[i].label
				}
			}
			right := ""
			if i < len(suggestions) {
				s := suggestions[i]
				streak := fmt.Sprintf("%d-day streak", s.StreakDays)
				right = fmt.Sprintf("%s  %-14s  %s", s.Date.Format("Mon Jan 02"), streak, s.Explanation)
			}
			fmt.Printf("    %-32s  %s\n", left, right)
		}
		fmt.Println()

		return nil
	},
}

func init() {
	suggestCmd.Flags().Int("year", time.Now().Year(), "Year to suggest PTO for")
	suggestCmd.Flags().Int("vacations", 2, "Max number of week-long vacations (5+ days); 0 for long weekends only")
	rootCmd.AddCommand(suggestCmd)
}
