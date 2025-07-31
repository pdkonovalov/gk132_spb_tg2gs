package config

import (
	"fmt"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v3"
)

const (
	LogLevelDebug = "debug"
	LogLevelProd  = "prod"
)

type Config struct {
	LogLevel string `yaml:"LogLevel" env:"LOG_LEVEL"`

	TelegramTimezone string `yaml:"TelegramTimezone" env:"TELEGRAM_TIMEZONE"`
	TelegramBotToken string `yaml:"TelegramBotToken" env:"TELEGRAM_BOT_TOKEN"`
	TelegramChatID   int64  `yaml:"TelegramChatID" env:"TELEGRAM_CHAT_ID"`

	GoogleSheetsServiceAccountCredentialsFile string `yaml:"GoogleSheetsServiceAccountCredentialsFile" env:"GOOGLE_SHEETS_SERVICE_ACCOUNT_CREDENTIALS_FILE"`
	GoogleSheetsSpreadsheetID                 string `yaml:"GoogleSheetsSpreadsheetID" env:"GOOGLE_SHEETS_SPREADSHEET_ID"`
	GoogleSheetsSheet                         string `yaml:"GoogleSheetsSheet" env:"GOOGLE_SHEETS_SHEET"`
}

func New() (*Config, error) {
	cfg := Config{}

	err := cleanenv.ReadConfig("config.yml", &cfg)
	if err != nil {
		err := cleanenv.ReadEnv(&cfg)
		if err != nil {
			return nil, err
		}
	}

	if cfg.LogLevel != LogLevelDebug && cfg.LogLevel != LogLevelProd {
		return nil, fmt.Errorf("Invalid LogLevel config variable value: '%s', must be %s or %s", cfg.LogLevel, LogLevelDebug, LogLevelProd)
	}

	return &cfg, nil
}

func (cfg *Config) StringSecureMasked() (string, error) {
	cfg_masked := new(Config)
	*cfg_masked = *cfg

	cfg_masked.TelegramBotToken = strings.Repeat("*", len(cfg_masked.TelegramBotToken))

	cfg_masked_yml, err := yaml.Marshal(cfg_masked)
	if err != nil {
		return "", fmt.Errorf("Error marshal config to yml: %s", err)
	}

	return string(cfg_masked_yml), nil
}
