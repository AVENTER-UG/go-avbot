package main

import (
	"github.com/AVENTER-UG/util/util"
)

func init() {
	BindAddress = util.Getenv("BIND_ADDRESS", "0.0.0.0:4050")
	DatabaseType = util.Getenv("DATABASE_TYPE", "sqlite3")
	DatabaseURL = util.Getenv("DATABASE_URL", "go-neb.db?_busy_timeout=5000")
	BaseURL = util.Getenv("BASE_URL", "http://localhost:4050/")
	ConfigFile = util.Getenv("CONFIG_FILE", "./data/config.yaml")
}
