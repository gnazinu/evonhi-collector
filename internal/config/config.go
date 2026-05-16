package config

import (
	"errors"
	"flag"
	"os"
	"time"
)

type Config struct {
	KubeconfigPath string
	ClusterID      string
	Endpoint       string
	Token          string
	HMACKey        string
	Pretty         bool
	DryRun         bool
	CollectTimeout time.Duration
	SendTimeout    time.Duration
	MaxElapsed     time.Duration
	// EnableSecretReading gates whether the agent calls the Kubernetes Secrets API.
	// Off by default (Zero-Trust): mount references are inferred from Pod specs only.
	EnableSecretReading bool
}

// Load reads CLI flags and env vars. Env wins as default; flag overrides.
func Load() (*Config, error) {
	c := &Config{}
	flag.StringVar(&c.KubeconfigPath, "kubeconfig", "", "optional kubeconfig path (fallback if not in-cluster)")
	flag.StringVar(&c.ClusterID, "cluster-id", getenv("EVONHI_CLUSTER_ID", "unknown"), "logical cluster identifier")
	flag.StringVar(&c.Endpoint, "endpoint", os.Getenv("EVONHI_ENDPOINT"), "telemetry endpoint URL (e.g. https://api.evonhi.io/api/v1/connector-ingest/telemetry)")
	flag.StringVar(&c.Token, "token", os.Getenv("EVONHI_TOKEN"), "connector token (sent as X-Connector-Token)")
	flag.StringVar(&c.HMACKey, "hmac-key", os.Getenv("EVONHI_HMAC_KEY"), "HMAC-SHA256 signing key (required when sending; off only in --dry-run)")
	flag.BoolVar(&c.Pretty, "pretty", false, "pretty-print JSON when dry-run")
	flag.BoolVar(&c.DryRun, "dry-run", false, "print payload to stdout instead of sending")
	flag.DurationVar(&c.CollectTimeout, "collect-timeout", durenv("EVONHI_COLLECT_TIMEOUT", 2*time.Minute), "collection timeout")
	flag.DurationVar(&c.SendTimeout, "send-timeout", durenv("EVONHI_SEND_TIMEOUT", 30*time.Second), "per-request HTTP timeout")
	flag.DurationVar(&c.MaxElapsed, "max-elapsed", durenv("EVONHI_MAX_ELAPSED", 10*time.Minute), "max total time for retries")
	flag.BoolVar(&c.EnableSecretReading, "enable-secret-reading", boolenv("EVONHI_ENABLE_SECRET_READING", false), "opt-in: call Secrets API for metadata (off by default; mounts are inferred from Pods)")
	flag.Parse()
	return c, c.validate()
}

func (c *Config) validate() error {
	if c.DryRun {
		return nil
	}
	if c.Endpoint == "" {
		return errors.New("endpoint required (--endpoint or EVONHI_ENDPOINT) — or use --dry-run")
	}
	if c.Token == "" {
		return errors.New("token required (--token or EVONHI_TOKEN) — or use --dry-run")
	}
	if c.HMACKey == "" {
		return errors.New("hmac key required (--hmac-key or EVONHI_HMAC_KEY) — or use --dry-run")
	}
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func boolenv(k string, def bool) bool {
	switch os.Getenv(k) {
	case "1", "true", "TRUE", "True", "yes", "on":
		return true
	case "0", "false", "FALSE", "False", "no", "off":
		return false
	}
	return def
}

func durenv(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
