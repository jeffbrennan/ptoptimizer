package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Holiday struct {
	Name string `json:"name"`
	Date string `json:"date"` // YYYY-MM-DD
}

type AccrualConfig struct {
	CurrentBalanceHours float64 `json:"current_balance_hours"`
	BalanceAsOfDate     string  `json:"balance_as_of_date"` // YYYY-MM-DD
	AccrualRateHours    float64 `json:"accrual_rate_hours"`  // hours per pay period
	PayPeriodDays       int     `json:"pay_period_days"`     // e.g. 14 for biweekly
	MaxBalanceHours     float64 `json:"max_balance_hours"`   // cap, 0 = unlimited
}

type PlannedPTO struct {
	StartDate string  `json:"start_date"`      // YYYY-MM-DD
	EndDate   string  `json:"end_date"`         // YYYY-MM-DD
	Hours     float64 `json:"hours"`            // hours used (default 8 per weekday)
	Label     string  `json:"label,omitempty"`
}

type Config struct {
	Holidays       []Holiday     `json:"holidays"`
	Accrual        AccrualConfig `json:"accrual"`
	PlannedTimeOff []PlannedPTO  `json:"planned_time_off"`
}

func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ptopt"), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path, err := ConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
