package server

import (
	"flag"
	"strings"

	"github.com/caarlos0/env/v11"
)

const (
	defaultServerAddress  = "http://localhost:8081"
	defaultAccrualAddress = "http://localhost:8080"
)

type Config struct {
	ServerAddress  string `env:"RUN_ADDRESS"`
	AccrualAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	DSN            string `env:"DATABASE_URI"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{
		ServerAddress:  defaultServerAddress,
		AccrualAddress: defaultAccrualAddress,
	}

	flag.StringVar(&cfg.ServerAddress, "a", defaultServerAddress, "server address default http://localhost:8081")
	flag.StringVar(&cfg.AccrualAddress, "r", defaultAccrualAddress, "accrual system address")
	flag.StringVar(&cfg.DSN, "d", "", "database connection string")

	flag.Parse()

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	if strings.HasPrefix(cfg.ServerAddress, "http://") {
		cfg.ServerAddress = strings.Replace(cfg.ServerAddress, "http://", "", -1)
	}

	return cfg, nil
}
