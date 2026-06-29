package config

import "os"

type Config struct {
	DatabaseURL             string
	Port                    string
	KafkaBrokers            string
	JitsiBaseURL            string
	JitsiApiKey             string
	JitsiAppID              string
	JitsiPrivateKeyFilename string
}

func Load() Config {
	return Config{
		DatabaseURL:             env("DATABASE_URL_GENERAL", "postgres://osship:osship_secret@postgres:5432/osship?sslmode=disable&search_path=general"),
		Port:                    env("PORT", "8084"),
		KafkaBrokers:            env("KAFKA_BROKERS", "kafka:9092"),
		JitsiBaseURL:            env("JITSI_BASE_URL", "https://meet.jit.si"),
		JitsiAppID:              env("JITSI_APP_ID", ""),
		JitsiApiKey:             env("JITSI_API_KEY", ""),
		JitsiPrivateKeyFilename: env("JITSI_PRIVATE_KEY_PATH", ""),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
