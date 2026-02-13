package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/jeffb/ptoptimizer/internal/config"
	"github.com/jeffb/ptoptimizer/internal/holidays"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup wizard for ptoptimizer",
	Long:  "Walks you through holiday selection and accrual configuration in one session.",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	year := time.Now().Year()
	allHolidays := holidays.USFederalHolidays(year)

	for {
		// Run main form (holiday select + accrual) with back navigation between groups
		selectedIndices, accrual, err := runMainForm(allHolidays)
		if err != nil {
			return err
		}

		selectedHolidays := make([]config.Holiday, len(selectedIndices))
		for i, idx := range selectedIndices {
			selectedHolidays[i] = allHolidays[idx]
		}

		// Custom holidays (dynamic loop, handled separately)
		customHolidays, err := addCustomHolidays()
		if err != nil {
			return err
		}
		selectedHolidays = append(selectedHolidays, customHolidays...)

		// Parse accrual strings into config
		accrualCfg := parseAccrual(accrual)

		// Confirmation — decline loops back to restart
		confirmed, err := confirmSave(selectedHolidays, accrualCfg)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Let's start over.\n")
			continue
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		cfg.Holidays = selectedHolidays
		cfg.Accrual = *accrualCfg

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Println("Configuration saved! You're all set.")
		fmt.Println("Try `ptopt suggest` to see recommended PTO days.")
		return nil
	}
}

type accrualInput struct {
	balanceStr string
	asOfDate   string
	rateStr    string
	periodDays int
	maxStr     string
}

// runMainForm runs holiday selection and accrual config as groups in a single form,
// giving the user Shift+Tab / back navigation between the two steps.
func runMainForm(allHolidays []config.Holiday) ([]int, *accrualInput, error) {
	// Holiday multiselect — all pre-selected
	options := make([]huh.Option[int], len(allHolidays))
	selected := make([]int, len(allHolidays))
	for i, h := range allHolidays {
		options[i] = huh.NewOption(fmt.Sprintf("%s (%s)", h.Name, h.Date), i)
		selected[i] = i
	}

	// Accrual fields with defaults
	today := time.Now().Format("2006-01-02")
	acc := &accrualInput{
		balanceStr: "0",
		asOfDate:   today,
		rateStr:    "0",
		maxStr:     "0",
	}

	form := huh.NewForm(
		// Group 1: Holidays
		huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title("Select your holidays").
				Description("All US federal holidays are pre-selected. Deselect any your employer doesn't observe.\nPress enter to continue, shift+tab to go back.").
				Options(options...).
				Value(&selected),
		),

		// Group 2: Accrual
		huh.NewGroup(
			huh.NewInput().
				Title("Current PTO balance (hours)").
				Value(&acc.balanceStr).
				Validate(validateHours("balance", 0, maxPTOHours)),
			huh.NewInput().
				Title("Balance as-of date (YYYY-MM-DD)").
				Value(&acc.asOfDate).
				Validate(validateDate),
			huh.NewInput().
				Title("Accrual rate (hours per pay period)").
				Value(&acc.rateStr).
				Validate(validateHours("accrual rate", 0, maxAccrualRate)),
			huh.NewSelect[int]().
				Title("Pay period").
				Options(
					huh.NewOption("Biweekly (every 2 weeks)", 14),
					huh.NewOption("Semi-monthly (twice a month)", 15),
					huh.NewOption("Monthly", 30),
					huh.NewOption("Weekly", 7),
				).
				Value(&acc.periodDays),
			huh.NewInput().
				Title("Maximum balance cap (hours, 0 = unlimited)").
				Value(&acc.maxStr).
				Validate(validateHours("max balance", 0, maxPTOHours)),
		),
	)

	if err := form.Run(); err != nil {
		return nil, nil, err
	}

	return selected, acc, nil
}

func addCustomHolidays() ([]config.Holiday, error) {
	var customs []config.Holiday

	var addCustom bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Add custom holidays?").
				Description("e.g. company holidays, floating holidays").
				Value(&addCustom),
		),
	)
	if err := form.Run(); err != nil {
		return nil, err
	}

	for addCustom {
		var name, date string
		var addAnother bool

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Holiday name").
					Value(&name),
				huh.NewInput().
					Title("Date (YYYY-MM-DD)").
					Value(&date).
					Validate(validateFutureDate),
				huh.NewConfirm().
					Title("Add another custom holiday?").
					Value(&addAnother),
			),
		)

		if err := form.Run(); err != nil {
			return nil, err
		}

		customs = append(customs, config.Holiday{Name: name, Date: date})
		addCustom = addAnother
	}

	return customs, nil
}

func parseAccrual(acc *accrualInput) *config.AccrualConfig {
	balance, _ := strconv.ParseFloat(acc.balanceStr, 64)
	rate, _ := strconv.ParseFloat(acc.rateStr, 64)
	max, _ := strconv.ParseFloat(acc.maxStr, 64)

	return &config.AccrualConfig{
		CurrentBalanceHours: balance,
		BalanceAsOfDate:     acc.asOfDate,
		AccrualRateHours:    rate,
		PayPeriodDays:       acc.periodDays,
		MaxBalanceHours:     max,
	}
}

func confirmSave(hols []config.Holiday, accrual *config.AccrualConfig) (bool, error) {
	var sb strings.Builder
	sb.WriteString("Holidays:\n")
	for _, h := range hols {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", h.Date, h.Name))
	}
	sb.WriteString(fmt.Sprintf("\nBalance:    %.1f hours (as of %s)\n", accrual.CurrentBalanceHours, accrual.BalanceAsOfDate))
	sb.WriteString(fmt.Sprintf("Accrual:    %.1f hours every %d days\n", accrual.AccrualRateHours, accrual.PayPeriodDays))
	if accrual.MaxBalanceHours > 0 {
		sb.WriteString(fmt.Sprintf("Cap:        %.1f hours\n", accrual.MaxBalanceHours))
	} else {
		sb.WriteString("Cap:        unlimited\n")
	}

	fmt.Println("\n" + sb.String())

	confirmed := true
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Save this configuration?").
				Description("Select 'No' to start over.").
				Affirmative("Yes").
				Negative("No").
				Value(&confirmed),
		),
	)
	if err := form.Run(); err != nil {
		return false, err
	}
	return confirmed, nil
}

const (
	maxPTOHours    = 480 // 60 days × 8 hours
	maxAccrualRate = 40  // 5 days per pay period is extremely generous
)

// validateHours returns a validator that checks a numeric string is within [min, max].
func validateHours(field string, min, max float64) func(string) error {
	return func(s string) error {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("enter a valid number")
		}
		if v < min {
			return fmt.Errorf("%s cannot be negative", field)
		}
		if v > max {
			return fmt.Errorf("%s cannot exceed %.0f hours", field, max)
		}
		return nil
	}
}

func validateDate(s string) error {
	_, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("invalid date format, use YYYY-MM-DD")
	}
	return nil
}

func validateFutureDate(s string) error {
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("invalid date format, use YYYY-MM-DD")
	}
	if d.Before(time.Now().Truncate(24 * time.Hour)) {
		return fmt.Errorf("date cannot be in the past")
	}
	return nil
}
