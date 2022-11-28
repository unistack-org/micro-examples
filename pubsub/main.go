package main

import (
	"context"
	"embed"
	"os"
	"os/signal"
	"syscall"
	"time"

	kgo "github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"

	//"github.com/golang-migrate/migrate/v4"
	//"github.com/golang-migrate/migrate/v4/database/sqlite"
	//"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/unistack-org/micro-examples/grpc/handler"
	bkgo "go.unistack.org/micro-broker-kgo/v3"
	jsoncodec "go.unistack.org/micro-codec-json/v3"
	protocodec "go.unistack.org/micro-codec-proto/v3"
	yamlcodec "go.unistack.org/micro-codec-yaml/v3"
	envconfig "go.unistack.org/micro-config-env/v3"
	fileconfig "go.unistack.org/micro-config-file/v3"
	vaultconfig "go.unistack.org/micro-config-vault/v3"
	prometheusmeter "go.unistack.org/micro-meter-prometheus/v3"
	httpsrv "go.unistack.org/micro-server-http/v3"
	otracer "go.unistack.org/micro-tracer-opentracing/v3"
	recoverywrapper "go.unistack.org/micro-wrapper-recovery/v3"
	requestidwrapper "go.unistack.org/micro-wrapper-requestid/v3"
	validatorwrapper "go.unistack.org/micro-wrapper-validator/v3"
	"go.unistack.org/micro/v3"
	"go.unistack.org/micro/v3/broker"
	"go.unistack.org/micro/v3/client"
	"go.unistack.org/micro/v3/config"
	"go.unistack.org/micro/v3/logger"
	loggerwrapper "go.unistack.org/micro/v3/logger/wrapper"
	"go.unistack.org/micro/v3/meter"
	meterhandler "go.unistack.org/micro/v3/meter/handler"
	meterwrapper "go.unistack.org/micro/v3/meter/wrapper"
	"go.unistack.org/micro/v3/server"
	health "go.unistack.org/micro/v3/server/health"
	healthhandler "go.unistack.org/micro/v3/server/health"
	"go.unistack.org/micro/v3/tracer"
	tracerwrapper "go.unistack.org/micro/v3/tracer/wrapper"
)

//go:embed migrations
var fs embed.FS

var (
	appName    = "grpc"
	BuildDate  string
	AppVersion string
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-ch
		logger.Infof(ctx, "handle signal %v, exiting", sig)
		cancel()
	}()

	cfg := newConfig(appName, AppVersion)
	if err := config.Load(ctx,
		[]config.Config{
			config.NewConfig(
				config.Struct(cfg),
			),
			envconfig.NewConfig(
				config.Struct(cfg),
			),
			fileconfig.NewConfig(
				config.Struct(cfg),
				config.Codec(yamlcodec.NewCodec()),
				config.AllowFail(true),
				fileconfig.Path("./local.yaml"), // nearby file
			),
			vaultconfig.NewConfig(
				config.Struct(cfg),
				config.Codec(yamlcodec.NewCodec()),
				config.AllowFail(true),
				config.BeforeLoad(func(ctx context.Context, c config.Config) error {
					return c.Init(
						vaultconfig.Address(cfg.Vault.Address),
						vaultconfig.Token(cfg.Vault.Token),
						vaultconfig.Path(cfg.Vault.Path),
					)
				}),
			),
		}, config.LoadOverride(true),
	); err != nil {
		logger.Fatalf(ctx, "failed to load config: %v", err)
	}

	meter.DefaultMeter = prometheusmeter.NewMeter()

	tracer.DefaultTracer = otracer.NewTracer()

	logger.Infof(ctx, "try to connect database")
	db, err := DatabaseConnect(cfg.Database)
	if err != nil {
		logger.Fatalf(ctx, "failed to connect to database: %v", err)
	}
	logger.Infof(ctx, "database connected")

	defer db.Close()

	/*
		if cfg.Database.MigrateUp {
			driver, err := sqlite.WithInstance(db.DB, &sqlite.Config{
				MigrationsTable: sqlite.DefaultMigrationsTable,
				DatabaseName:    cfg.Database.Name,
			})
			if err != nil {
				logger.Fatalf(ctx, "failed to run database migrations: %v", err)
			}
			source, err := iofs.New(fs, "migrations/sqlite")
			if err != nil {
				logger.Fatalf(ctx, "failed to run database migrations: %v", err)
			}

			m, err := migrate.NewWithInstance("fs", source, "authn", driver)
			if err != nil {
				logger.Fatalf(ctx, "failed to run database migrations: %v", err)
			}

			if err = m.Up(); err != nil && err != migrate.ErrNoChange {
				logger.Fatalf(ctx, "failed to run database migrations: %v", err)
			}
		}

		if cfg.Database.MigrateDown {
			driver, err := sqlite.WithInstance(db.DB, &sqlite.Config{
				MigrationsTable: sqlite.DefaultMigrationsTable,
				DatabaseName:    cfg.Database.Name,
			})
			if err != nil {
				logger.Fatalf(ctx, "failed to run database migrations: %v", err)
			}
			source, err := iofs.New(fs, "migrations/sqlite")
			if err != nil {
				logger.Fatalf(ctx, "failed to run database migrations: %v", err)
			}

			// TODO: pass own logger
			m, err := migrate.NewWithInstance("fs", source, "authn", driver)
			if err != nil {
				logger.Fatalf(ctx, "failed to run database migrations: %v", err)
			}

			if err = m.Down(); err != nil && err != migrate.ErrNoChange {
				logger.Fatalf(ctx, "failed to run database migrations: %v", err)
			}
		}
	*/
	if cfg.Server.Name != "" {
		appName = cfg.Server.Name
	}

	kopts := []kgo.Opt{
		kgo.FetchMaxWait(1 * time.Second),
		kgo.StopProducerOnDataLossDetected(),
		kgo.ClientID(cfg.Server.Name),
		kgo.ProducerBatchCompression(kgo.NoCompression()),
		kgo.MaxBufferedRecords(1000),
		kgo.RequiredAcks(kgo.LeaderAck()),
		kgo.ProducerBatchMaxBytes(1 * 1024 * 1024),
		kgo.ProducerLinger(200 * time.Microsecond),
		kgo.ProducerBatchMaxBytes(4 * 1024 * 1024),
		kgo.FetchMaxWait(1 * time.Second),
		kgo.FetchMaxBytes(1 * 1024 * 1024),
	}
	if len(cfg.Broker.Login) > 0 && len(cfg.Broker.Passw) > 0 {
		kopts = append(kopts,
			kgo.SASL((plain.Auth{User: cfg.Broker.Login, Pass: cfg.Broker.Passw}).AsMechanism()),
		)
	}

	b := bkgo.NewBroker(
		broker.Addrs(cfg.Broker.Address...),
		broker.Codec(jsoncodec.NewCodec()),
		broker.Context(ctx),
		bkgo.CommitInterval(1*time.Second),
		bkgo.Options(kopts...),
	)
	if err = b.Init(); err != nil {
		logger.Fatalf(ctx, "broker init err: %v", err)
	}

	svc := micro.NewService(
		micro.Context(ctx),
		micro.Server(server.NewServer(server.Broker(b))),
		micro.Client(client.NewClient(client.Broker(b))),
		micro.Broker(b),
	)

	if err = svc.Init(
		micro.Name(appName),
		micro.Version(AppVersion),
	); err != nil {
		logger.Fatalf(ctx, "service init err: %v", err)
	}

	if err = svc.Server().Init(
		server.Name(cfg.Server.Name),
		server.Version(cfg.Server.Version),
		server.Address(cfg.Server.Address),
		server.Codec("application/grpc", protocodec.NewCodec()),
		server.Codec("application/grpc+proto", protocodec.NewCodec()),
		server.Context(ctx),
		server.WrapSubscriber(recoverywrapper.NewServerSubscriberWrapper()),
		server.WrapSubscriber(requestidwrapper.NewServerSubscriberWrapper()),
		server.WrapSubscriber(loggerwrapper.NewServerSubscriberWrapper(
			loggerwrapper.WithLogger(logger.DefaultLogger),
			loggerwrapper.WithLevel(logger.InfoLevel),
		)),
		server.WrapSubscriber(meterwrapper.NewServerSubscriberWrapper(
			meterwrapper.ServiceName(svc.Server().Options().Name),
			meterwrapper.ServiceVersion(svc.Server().Options().Version),
			meterwrapper.ServiceID(svc.Server().Options().ID),
		)),
		server.WrapSubscriber(tracerwrapper.NewServerSubscriberWrapper(
			tracerwrapper.WithTracer(tracer.DefaultTracer),
		)),
		server.WrapSubscriber(validatorwrapper.NewServerSubscriberWrapper()),
	); err != nil {
		logger.Fatalf(ctx, "server init err: %v", err)
	}

	if err = svc.Client().Init(
		client.ContentType("application/grpc"),
		client.Codec("application/grpc", protocodec.NewCodec()),
		client.Codec("application/grpc+proto", protocodec.NewCodec()),
		client.Wrap(requestidwrapper.NewClientWrapper()),
		client.Wrap(loggerwrapper.NewClientWrapper(
			loggerwrapper.WithLogger(logger.DefaultLogger),
			loggerwrapper.WithLevel(logger.InfoLevel),
		)),
		client.Wrap(meterwrapper.NewClientWrapper(
			meterwrapper.ServiceName(svc.Server().Options().Name),
			meterwrapper.ServiceVersion(svc.Server().Options().Version),
			meterwrapper.ServiceID(svc.Server().Options().ID),
		)),
		client.Wrap(tracerwrapper.NewClientWrapper(
			tracerwrapper.WithTracer(tracer.DefaultTracer),
		)),
		client.Wrap(validatorwrapper.NewClientWrapper()),
	); err != nil {
		logger.Fatalf(ctx, "client init err: %v", err)
	}

	h, err := handler.NewHandler(
		db,
	)
	if err != nil {
		logger.Fatalf(ctx, "handler init failed: %v", err)
	}

	if err = health.RegisterHealthServer(svc.Server(), health.NewHandler()); err != nil {
		logger.Fatalf(ctx, "failed to register health handler: %v", err)
	}

	hsvc := httpsrv.NewServer(
		server.Codec("application/json", jsoncodec.NewCodec()),
		server.Address(cfg.Meter.Address),
		server.Context(ctx),
	)

	if err = hsvc.Init(); err != nil {
		logger.Fatalf(ctx, "failed to init http srv: %v", err)
	}

	if err := healthhandler.RegisterHealthServer(hsvc, healthhandler.NewHandler()); err != nil {
		logger.Fatalf(ctx, "failed to register http health handler: %v", err)
	}
	if err := meterhandler.RegisterMeterServer(hsvc, meterhandler.NewHandler()); err != nil {
		logger.Fatalf(ctx, "failed to register http meter handler: %v", err)
	}
	if err = hsvc.Start(); err != nil {
		logger.Fatalf(ctx, "failed to run http srv: %v", err)
	}

	if err := micro.RegisterSubscriber(cfg.App.Topic, svc.Server(), h,
		server.SubscriberGroup(appName),
		server.SubscriberAck(true),
	); err != nil {
		logger.Fatalf(ctx, "failed to register pubsub handler: %v", err)
	}

	if err := svc.Run(); err != nil {
		logger.Fatalf(ctx, "svc run err: %v", err)
	}
}
