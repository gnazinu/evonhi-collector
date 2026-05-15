package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"github.com/tu-usuario/evonhi-collector/internal/k8s"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "optional path to kubeconfig (fallback when not in-cluster)")
	clusterID := flag.String("cluster-id", envOrDefault("EVONHI_CLUSTER_ID", "unknown"), "logical cluster identifier")
	pretty := flag.Bool("pretty", true, "pretty-print JSON output")
	timeout := flag.Duration("timeout", 2*time.Minute, "collection timeout")
	flag.Parse()

	cs, err := k8s.BuildClient(*kubeconfig)
	if err != nil {
		log.Fatalf("k8s client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	payload, err := k8s.Collect(ctx, cs, *clusterID)
	if err != nil {
		log.Fatalf("collect: %v", err)
	}

	enc := json.NewEncoder(os.Stdout)
	if *pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(payload); err != nil {
		log.Fatalf("encode: %v", err)
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
