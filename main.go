package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/pdkonovalov/gk132_spb_tg2gs/internal/config"
	"github.com/pdkonovalov/gk132_spb_tg2gs/internal/interfaces/telegram"
	"github.com/pdkonovalov/gk132_spb_tg2gs/internal/repository/google_sheets"
)

func main() {
	log.Print("read configuration...")

	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Error init config: %s", err)
	}

	log.Print("configuration successfull loaded:")
	cfg_masked, err := cfg.StringSecureMasked()
	if err != nil {
		log.Fatalf("Error print config: %s", err)
	}

	log.Print("\n" + cfg_masked)

	log.Print("connect to google sheets...")

	repo, err := google_sheets.New(cfg)
	if err != nil {
		log.Fatalf("Error init google sheets: %s", err)
	}

	log.Print("connect to google sheets successfull")

	log.Print("configure telegram client...")

	telegram_client, err := telegram.New(cfg, repo)
	if err != nil {
		log.Fatalf("Error configure telegram client: %s", err)
	}

	log.Print("configure telegram client successfull")
	log.Print("starting telegram client...")

	err = telegram_client.Run()
	if err != nil {
		log.Fatalf("Error start telegram client: %s", err)
	}

	log.Print("telegram client successfull started")
	log.Print("press ctrl c to shutdown")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		<-ctx.Done()

		log.Print("shutdown telegram client...")
		telegram_client.Stop()
		if err != nil {
			log.Printf("failed shutdown telegram client: %s", err)
		}

		log.Print("close google sheets connection...")
		err := repo.Close(ctx)
		if err != nil {
			log.Printf("failed close google sheets connection: %s", err)
		}
	}()
	wg.Wait()
	log.Print("exit")
}
