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

	log.Print("configure bot...")

	bot, err := telegram.New(cfg, repo)
	if err != nil {
		log.Fatalf("Error init bot: %s", err)
	}

	log.Print("configure bot successfull")
	log.Print("starting bot...")

	go bot.Start()

	log.Print("bot successfull started")
	log.Print("press ctrl c to shutdown")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		<-ctx.Done()

		log.Print("shutdown bot...")
		bot.Stop()

		log.Print("close google sheets connection...")
		err := repo.Close(ctx)
		if err != nil {
			log.Printf("failed close google sheets connection: %s", err)
		}
	}()
	wg.Wait()
	log.Print("exit")
}
