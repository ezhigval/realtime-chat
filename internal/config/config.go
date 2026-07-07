package config

import (
	"os"

	"github.com/ezhigval/go-toolkit/config"
)

type Config struct {
	Port         string `env:"PORT" envDefault:"8086"`
	InstanceID   string `env:"INSTANCE_ID"`
	LogLevel     string `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat    string `env:"LOG_FORMAT" envDefault:"json"`
	DatabaseURL  string `env:"DATABASE_URL,required"`
	RedisAddr    string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	RedisPassword string `env:"REDIS_PASSWORD"`
	RedisDB      int    `env:"REDIS_DB" envDefault:"0"`
}

func MustLoad() Config {
	cfg := config.MustLoad[Config]()
	if cfg.InstanceID == "" {
		host, _ := os.Hostname()
		cfg.InstanceID = host
	}
	return cfg
}
