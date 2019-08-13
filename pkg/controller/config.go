package controller

import "github.com/zh168654/Redis-Operator/pkg/config"

// Config contains the Controller settings
type Config struct {
	NbWorker int
	redis    config.Redis
}

// NewConfig builds and returns new Config instance
func NewConfig(nbWorker int, redis config.Redis) *Config {
	return &Config{
		NbWorker: nbWorker,
		redis:    redis,
	}
}
