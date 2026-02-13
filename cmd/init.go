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

	// Step 1: Holiday selection
	selectedHolidays, err := selectHolidays(allHolidays)
	if err != nil {
		return err
	}

	// Step 1b: Custom holidays
	customHolidays, err := addCustomHolidays()
	if err != nil {
		return err
	}
	selectedHolidays = append(selectedHolidays, customHolidays...)

	// Step 2: Accrual configuration
	accrual, err := configureAccrual()
	if err != nil {
		return err
	}

	// Step 3: Confirmation
	confirmed, err := confirmAndSave(selectedHolidays, accrual)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Setup cancelled.")
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	cfg.Holidays = selectedHolidays
	cfg.Accrual = *accrual

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println("Configuration saved! You're all set.")
	fmt.Println("Try `ptopt suggest` to see recommended PTO days.")
	return nil
}

func selectHolidays(allHolidays []config.Holiday) ([]config.Holiday, error) {
	// Build options — all pre-selected by default
	options := make([]huh.Option[int], len(allHolidays))
	selected := make([]int, len(allHolidays))
	for i, h := range allHolidays {
		options[i] = huh.NewOption(fmt.Sprintf("%s (%s)", h.Name, h.Date), i)
		selected[i] = i
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title("Select your holidays").
				Description("All US federal holidays are pre-selected. Deselect any your employer doesn't observe.").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	result := make([]config.Holiday, len(selected))
	for i, idx := range selected {
		result[i] = allHolidays[idx]
	}
	return result, nil
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
					Validate(func(s string) error {
						_, err := time.Parse("2006-01-02", s)
						if err != nil {
							return fmt.Errorf("invalid date format, use YYYY-MM-DD")
						}
						return nil
					}),
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

func configureAccrual() (*config.AccrualConfig, error) {
	var balanceStr, asOfDate, rateStr, maxStr string
	var periodDays int

	today := time.Now().Format("2006-01-02")

	balanceStr = "0"
	asOfDate = today
	rateStr = "0"
	maxStr = "0"

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Current PTO balance (hours)").
				Value(&balanceStr).
				Validate(validateFloat),
			huh.NewInput().
				Title("Balance as-of date").
				Description("YYYY-MM-DD").
				Value(&asOfDate).
				Validate(func(s string) error {
					_, err := time.Parse("2006-01-02", s)
					if err != nil {
						return fmt.Errorf("invalid date format, use YYYY-MM-DD")
					}
					return nil
				}),
			huh.NewInput().
				Title("Accrual rate (hours per pay period)").
				Value(&rateStr).
				Validate(validateFloat),
			huh.NewSelect[int]().
				Title("Pay period").
				Options(
					huh.NewOption("Biweekly (every 2 weeks)", 14),
					huh.NewOption("Semi-monthly (twice a month)", 15),
					huh.NewOption("Monthly", 30),
					huh.NewOption("Weekly", 7),
				).
				Value(&periodDays),
			huh.NewInput().
				Title("Maximum balance cap (hours, 0 = unlimited)").
				Value(&maxStr).
				Validate(validateFloat),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	balance, _ := strconv.ParseFloat(balanceStr, 64)
	rate, _ := strconv.ParseFloat(rateStr, 64)
	max, _ := strconv.ParseFloat(maxStr, 64)

	return &config.AccrualConfig{
		CurrentBalanceHours: balance,
		BalanceAsOfDate:     asOfDate,
		AccrualRateHours:    rate,
		PayPeriodDays:       periodDays,
		MaxBalanceHours:     max,
	}, nil
}

func confirmAndSave(hols []config.Holiday, accrual *config.AccrualConfig) (bool, error) {
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

	var confirmed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Save this configuration?").
				Value(&confirmed),
		),
	)
	if err := form.Run(); err != nil {
		return false, err
	}
	return confirmed, nil
}

func validateFloat(s string) error {
	_, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("enter a valid number")
	}
	return nil
}
