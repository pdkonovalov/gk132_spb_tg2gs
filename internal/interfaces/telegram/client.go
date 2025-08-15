package telegram

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pdkonovalov/gk132_spb_tg2gs/internal/config"
	"github.com/pdkonovalov/gk132_spb_tg2gs/internal/domain/repository"
	"github.com/pdkonovalov/gk132_spb_tg2gs/internal/parser"

	pebbledb "github.com/cockroachdb/pebble"
	boltstor "github.com/gotd/contrib/bbolt"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/contrib/middleware/ratelimit"
	"github.com/gotd/contrib/pebble"
	"github.com/gotd/contrib/storage"
	"github.com/gotd/td/examples"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/message/peer"
	"github.com/gotd/td/telegram/query"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"
	"go.etcd.io/bbolt"
	"golang.org/x/time/rate"
)

func isValidPhoneNumber(phone string) bool {
	e164Regex := `^\+[1-9]\d{1,14}$`
	re := regexp.MustCompile(e164Regex)
	phone = strings.ReplaceAll(phone, " ", "")

	return re.Find([]byte(phone)) != nil
}

func sessionFolder(phone string) string {
	var out []rune
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			out = append(out, r)
		}
	}
	return "phone-" + string(out)
}

type Client struct {
	ctx             context.Context
	cancel          context.CancelFunc
	waiter          *floodwait.Waiter
	client          *telegram.Client
	flow            auth.Flow
	peerDB          *pebble.PeerStorage
	api             *tg.Client
	updatesRecovery *updates.Manager
}

func New(cfg *config.Config, repo repository.Repository) (*Client, error) {
	location, err := time.LoadLocation(cfg.TelegramTimezone)
	if err != nil {
		return nil, fmt.Errorf("Failed load timezone: %s", err)
	}

	if ok := isValidPhoneNumber(cfg.TelegramPhone); !ok {
		return nil, fmt.Errorf("Invalid telegram phone number in config: %s", cfg.TelegramPhone)
	}

	sessionDir := filepath.Join("data/telegram/session", sessionFolder(cfg.TelegramPhone))
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, fmt.Errorf("Failed create session storage: %s", err)
	}

	sessionStorage := &telegram.FileSessionStorage{
		Path: filepath.Join(sessionDir, "session.json"),
	}

	db, err := pebbledb.Open(filepath.Join(sessionDir, "peers.pebble.db"), &pebbledb.Options{})
	if err != nil {
		return nil, fmt.Errorf("Failed create pebble storage: %s", err)
	}
	peerDB := pebble.NewPeerStorage(db)

	dispatcher := tg.NewUpdateDispatcher()

	updateHandler := storage.UpdateHook(dispatcher, peerDB)

	boltdb, err := bbolt.Open(filepath.Join(sessionDir, "updates.bolt.db"), 0666, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed create bolt storage: %s", err)
	}

	updatesRecovery := updates.New(updates.Config{
		Handler: updateHandler,
		Storage: boltstor.NewStateStorage(boltdb),
	})

	waiter := floodwait.NewWaiter().WithCallback(func(ctx context.Context, wait floodwait.FloodWait) {
		log.Print("Got FLOOD_WAIT. Will retry after", wait.Duration)
	})

	options := telegram.Options{
		SessionStorage: sessionStorage,
		UpdateHandler:  updatesRecovery,
		Middlewares: []telegram.Middleware{
			waiter,
			ratelimit.New(rate.Every(time.Millisecond*100), 5),
		},
		Device: telegram.DeviceConfig{},
	}
	client := telegram.NewClient(cfg.TelegramAppID, cfg.TelegramAppHash, options)
	api := client.API()

	resolver := storage.NewResolverCache(peer.Plain(api), peerDB)
	_ = resolver

	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewMessage) error {
		msg, ok := u.Message.(*tg.Message)
		if !ok {
			return nil
		}

		peer_chat, ok := msg.GetPeerID().(*tg.PeerUser)
		if !ok {
			return nil
		}

		if peer_chat.UserID != cfg.TelegramChatID {
			log.Printf("Ignoring message '%s' from chat %v", msg.Message, peer_chat.UserID)
			return nil
		}

		problem, ok := parser.ParseProblemMessage(msg.Message, location)
		if !ok {
			log.Printf("Message '%s' from target chat is not valid problem message", msg.Message)
			return nil
		}

		err := repo.Create(problem)
		if err != nil {
			return fmt.Errorf("Failed write problem '%s' to google sheets: %s", msg.Message, err)
		}

		log.Printf("Problem '%s' successfully writed to google sheets", msg.Message)

		return nil
	})

	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewChannelMessage) error {
		msg, ok := u.Message.(*tg.Message)
		if !ok {
			return nil
		}

		peer_chanel, ok := msg.GetPeerID().(*tg.PeerChannel)
		if !ok {
			return nil
		}

		if peer_chanel.ChannelID != cfg.TelegramChatID {
			log.Printf("Ignoring message '%s' from chat %v", msg.Message, peer_chanel.ChannelID)
			return nil
		}

		problem, ok := parser.ParseProblemMessage(msg.Message, location)
		if !ok {
			log.Printf("Message '%s' from target chat is not valid problem message", msg.Message)
			return nil
		}

		if problem == nil {
			return fmt.Errorf("Message '%s' is valid problem message, but parsed problem is nil", msg.Message)
		}

		if !problem.IsResolved {
			err := repo.Create(problem)
			if err != nil {
				return fmt.Errorf("Failed write problem '%s' to google sheets: %s", msg.Message, err)
			}
		} else {
			err := repo.Update(problem)
			if err != nil {
				return fmt.Errorf("Failed write problem '%s' to google sheets: %s", msg.Message, err)
			}
		}

		log.Printf("Problem '%s' successfully writed to google sheets", msg.Message)

		return nil
	})

	flow := auth.NewFlow(examples.Terminal{PhoneNumber: cfg.TelegramPhone}, auth.SendCodeOptions{})

	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		ctx:             ctx,
		cancel:          cancel,
		waiter:          waiter,
		client:          client,
		flow:            flow,
		peerDB:          peerDB,
		api:             api,
		updatesRecovery: updatesRecovery,
	}, nil
}

func (c *Client) Run() error {
	return c.waiter.Run(c.ctx, func(ctx context.Context) error {
		if err := c.client.Run(ctx, func(ctx context.Context) error {
			if err := c.client.Auth().IfNecessary(ctx, c.flow); err != nil {
				return err
			}

			self, err := c.client.Self(ctx)
			if err != nil {
				return err
			}

			log.Printf("Successfull logged in: %s %s %s %v", self.FirstName, self.LastName, self.Username, self.ID)

			collector := storage.CollectPeers(c.peerDB)
			if err := collector.Dialogs(ctx, query.GetDialogs(c.api).Iter()); err != nil {
				return err
			}

			fmt.Println("Listening for updates. Interrupt (Ctrl+C) to stop.")
			return c.updatesRecovery.Run(ctx, c.api, self.ID, updates.AuthOptions{
				IsBot: self.Bot,
				OnStart: func(ctx context.Context) {
					fmt.Println("Update recovery initialized and started, listening for events")
				},
			})
		}); err != nil {
			return err
		}
		return nil
	})
}

func (c *Client) Stop() error {
	c.cancel()
	return nil
}
