package main

import (
	"log"

	"github.com/caarlos0/env/v10"
)

type Config struct {
	MastodonServerDomain string `env:"MASTODON_SERVER_DOMAIN,required"`
	MastodonBearerToken  string `env:"MASTODON_BEARER_TOKEN,required"`
	DbName               string `env:"DB_NAME" envDefault:"data/local.db"`
	Address              string `env:"ADDRESS" envDefault:"0.0.0.0:8000"`
}

func GetConfig() Config {

	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("%+v\n", err)
	}
	return cfg
}
