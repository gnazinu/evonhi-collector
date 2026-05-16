package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/tu-usuario/evonhi-collector/internal/config"
	"github.com/tu-usuario/evonhi-collector/internal/k8s"
	"github.com/tu-usuario/evonhi-collector/internal/sender"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	cs, err := k8s.BuildClient(cfg.KubeconfigPath)
	if err != nil {
		log.Fatalf("k8s client: %v", err)
	}

	collectCtx, cancelCollect := context.WithTimeout(context.Background(), cfg.CollectTimeout)
	defer cancelCollect()

	payload, err := k8s.Collect(collectCtx, cs, cfg.ClusterID, cfg.EnableSecretReading)
	if err != nil {
		log.Fatalf("collect: %v", err)
	}

	env := sender.BuildEnvelope(payload)

	if cfg.DryRun {
		enc := json.NewEncoder(os.Stdout)
		if cfg.Pretty {
			enc.SetIndent("", "  ")
		}
		if err := enc.Encode(env); err != nil {
			log.Fatalf("encode: %v", err)
		}
		return
	}

	client := sender.NewClient(cfg.Endpoint, cfg.Token, cfg.HMACKey, cfg.SendTimeout, cfg.MaxElapsed)
	sendCtx, cancelSend := context.WithTimeout(context.Background(), cfg.MaxElapsed)
	defer cancelSend()
	if err := client.Send(sendCtx, env); err != nil {
		log.Fatalf("send: %v", err)
	}
}
