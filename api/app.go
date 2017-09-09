package api

import (
	"time"

	"github.com/go-redis/redis"
	"github.com/keratin/authn-server/config"
	"github.com/keratin/authn-server/data"
	"github.com/pkg/errors"

	"github.com/keratin/authn-server/data/mock"
	dataRedis "github.com/keratin/authn-server/data/redis"
)

type pinger func() bool

type App struct {
	DbCheck           pinger
	RedisCheck        pinger
	Config            *config.Config
	AccountStore      data.AccountStore
	RefreshTokenStore data.RefreshTokenStore
	KeyStore          data.KeyStore
	Actives           data.Actives
}

func NewApp() (*App, error) {
	cfg := config.ReadEnv()

	db, accountStore, err := data.NewDB(cfg.DatabaseURL)
	if err != nil {
		return nil, errors.Wrap(err, "NewDB")
	}

	opts, err := redis.ParseURL(cfg.RedisURL.String())
	if err != nil {
		return nil, errors.Wrap(err, "ParseURL")
	}
	redis := redis.NewClient(opts)

	tokenStore := &dataRedis.RefreshTokenStore{
		Client: redis,
		TTL:    cfg.RefreshTokenTTL,
	}

	var keyStore data.KeyStore
	if cfg.IdentitySigningKey == nil {
		keyStore, err = dataRedis.NewKeyStore(
			redis,
			cfg.AccessTokenTTL,
			time.Duration(500)*time.Millisecond,
			cfg.DBEncryptionKey,
		)
		if err != nil {
			return nil, errors.Wrap(err, "NewKeyStore")
		}
	} else {
		keyStore = mock.NewKeyStore(cfg.IdentitySigningKey)
	}

	actives := dataRedis.NewActives(
		redis,
		cfg.StatisticsTimeZone,
		cfg.DailyActivesRetention,
		cfg.WeeklyActivesRetention,
		5*12,
	)

	return &App{
		DbCheck:           func() bool { return db.Ping() == nil },
		RedisCheck:        func() bool { return redis.Ping().Err() == nil },
		Config:            cfg,
		AccountStore:      accountStore,
		RefreshTokenStore: tokenStore,
		KeyStore:          keyStore,
		Actives:           actives,
	}, nil
}
