package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/chekist32/goipay/internal/dto"
	handler_v1 "github.com/chekist32/goipay/internal/handler/v1"
	pb_v1 "github.com/chekist32/goipay/internal/pb/v1"
	"github.com/chekist32/goipay/internal/processor"
	"github.com/chekist32/goipay/internal/util"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v3"
)

type CliOpts struct {
	ConfigPath        string
	ClientCAPaths     string
	ReflectionEnabled bool
}

type TlsMode string

const (
	NONE_TLS_MODE TlsMode = "none"
	TLS_TLS_MODE  TlsMode = "tls"
	MTLS_TLS_MODE TlsMode = "mtls"
)

type AppConfigDaemon struct {
	Url  string `yaml:"url"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

type AppConfigTls struct {
	Mode string `yaml:"mode"`
	Ca   string `yaml:"ca"`
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

type AppConfig struct {
	Server struct {
		Host string       `yaml:"host"`
		Port string       `yaml:"port"`
		Tls  AppConfigTls `yaml:"tls"`
	} `yaml:"server"`

	Database struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
		User string `yaml:"user"`
		Pass string `yaml:"pass"`
		Name string `yaml:"name"`
	} `yaml:"database"`

	Coin struct {
		Xmr struct {
			Daemon AppConfigDaemon `yaml:"daemon"`
		} `yaml:"xmr"`
		Btc struct {
			Daemon AppConfigDaemon `yaml:"daemon"`
		} `yaml:"btc"`
		Ltc struct {
			Daemon AppConfigDaemon `yaml:"daemon"`
		} `yaml:"ltc"`
		Eth struct {
			Daemon AppConfigDaemon `yaml:"daemon"`
		} `yaml:"eth"`
		Bnb struct {
			Daemon AppConfigDaemon `yaml:"daemon"`
		} `yaml:"bnb"`
	} `yaml:"coin"`
}

func NewAppConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var conf AppConfig
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}

	conf.Server.Host = os.ExpandEnv(conf.Server.Host)
	conf.Server.Port = os.ExpandEnv(conf.Server.Port)

	conf.Server.Tls.Mode = os.ExpandEnv(conf.Server.Tls.Mode)
	conf.Server.Tls.Ca = os.ExpandEnv(conf.Server.Tls.Ca)
	conf.Server.Tls.Cert = os.ExpandEnv(conf.Server.Tls.Cert)
	conf.Server.Tls.Key = os.ExpandEnv(conf.Server.Tls.Key)

	conf.Database.Host = os.ExpandEnv(conf.Database.Host)
	conf.Database.Port = os.ExpandEnv(conf.Database.Port)
	conf.Database.User = os.ExpandEnv(conf.Database.User)
	conf.Database.Pass = os.ExpandEnv(conf.Database.Pass)
	conf.Database.Name = os.ExpandEnv(conf.Database.Name)

	conf.Coin.Xmr.Daemon.Url = os.ExpandEnv(conf.Coin.Xmr.Daemon.Url)
	conf.Coin.Xmr.Daemon.User = os.ExpandEnv(conf.Coin.Xmr.Daemon.User)
	conf.Coin.Xmr.Daemon.Pass = os.ExpandEnv(conf.Coin.Xmr.Daemon.Pass)

	conf.Coin.Btc.Daemon.Url = os.ExpandEnv(conf.Coin.Btc.Daemon.Url)
	conf.Coin.Btc.Daemon.User = os.ExpandEnv(conf.Coin.Btc.Daemon.User)
	conf.Coin.Btc.Daemon.Pass = os.ExpandEnv(conf.Coin.Btc.Daemon.Pass)

	conf.Coin.Ltc.Daemon.Url = os.ExpandEnv(conf.Coin.Ltc.Daemon.Url)
	conf.Coin.Ltc.Daemon.User = os.ExpandEnv(conf.Coin.Ltc.Daemon.User)
	conf.Coin.Ltc.Daemon.Pass = os.ExpandEnv(conf.Coin.Ltc.Daemon.Pass)

	conf.Coin.Eth.Daemon.Url = os.ExpandEnv(conf.Coin.Eth.Daemon.Url)
	conf.Coin.Eth.Daemon.User = os.ExpandEnv(conf.Coin.Eth.Daemon.User)
	conf.Coin.Eth.Daemon.Pass = os.ExpandEnv(conf.Coin.Eth.Daemon.Pass)

	conf.Coin.Bnb.Daemon.Url = os.ExpandEnv(conf.Coin.Bnb.Daemon.Url)
	conf.Coin.Bnb.Daemon.User = os.ExpandEnv(conf.Coin.Bnb.Daemon.User)
	conf.Coin.Bnb.Daemon.Pass = os.ExpandEnv(conf.Coin.Bnb.Daemon.Pass)

	return &conf, nil
}

type App struct {
	ctxCancel context.CancelFunc

	config *AppConfig
	opts   *CliOpts
	log    *zerolog.Logger

	dbConnPool       *pgxpool.Pool
	paymentProcessor *processor.PaymentProcessor
}

func (a *App) Start(ctx context.Context) error {
	if err := a.dbConnPool.Ping(ctx); err != nil {
		a.log.Info().Err(err).Msg("Failed to connect to the database.")
		return err
	}
	defer a.dbConnPool.Close()

	lis, err := net.Listen("tcp", a.config.Server.Host+":"+a.config.Server.Port)
	if err != nil {
		a.log.Info().Err(err).Msgf("Failed to listen on port %v.", a.config.Server.Port)
		return err
	}

	g := grpc.NewServer(getGrpcServerOptions(a)...)
	pb_v1.RegisterUserServiceServer(g, handler_v1.NewUserGrpc(a.dbConnPool, a.log))
	pb_v1.RegisterInvoiceServiceServer(g, handler_v1.NewInvoiceGrpc(a.dbConnPool, a.paymentProcessor, a.log))

	if a.opts.ReflectionEnabled {
		reflection.Register(g)
	}

	h := health.NewServer()
	grpc_health_v1.RegisterHealthServer(g, h)
	h.SetServingStatus("", grpc_health_v1.HealthCheckResponse_UNKNOWN)
	go func() {
		for {
			select {
			case <-time.After(util.HEALTH_CHECK_TIEMOUT):
				s := grpc_health_v1.HealthCheckResponse_SERVING
				if err := a.dbConnPool.Ping(ctx); err != nil {
					a.log.Debug().Err(err).Msg("The database health check failed.")
					s = grpc_health_v1.HealthCheckResponse_NOT_SERVING
				}
				h.SetServingStatus("", s)
			case <-ctx.Done():
				return
			}
		}
	}()

	ch := make(chan error, 1)
	go func() {
		if err := g.Serve(lis); err != nil {
			a.log.Info().Err(err).Msg("Failed to start the server.")
			ch <- err
		}
		close(ch)
	}()

	a.log.Info().Msgf("Starting server %v\n", lis.Addr())

	select {
	case err = <-ch:
		return err
	case <-ctx.Done():
		a.ctxCancel()
		g.GracefulStop()
		return nil
	}
}

func getGrpcServerOptions(a *App) []grpc.ServerOption {
	grpcOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			NewMetadataInterceptor(a.log).Intercepte,
			NewRequestLoggingInterceptor(a.log).Intercepte,
		),
	}

	if creds, enabled := getGrpcCrednetials(a.log, a.config, a.opts); enabled {
		grpcOpts = append(grpcOpts, creds)
	}

	return grpcOpts
}

func getMtlsCofig(log *zerolog.Logger, c *AppConfig, opts *CliOpts) *tls.Config {
	config := getTlsConfig(log, c)

	if strings.TrimSpace(opts.ClientCAPaths) == "" {
		log.Fatal().Msg("-client-ca must specify at least one path")
	}

	paths := strings.Split(strings.TrimSpace(opts.ClientCAPaths), ",")
	if len(paths) == 0 {
		log.Fatal().Msg("-client-ca must specify at least one path")
	}

	certPool := x509.NewCertPool()
	for i := 0; i < len(paths); i++ {
		trustedCert, err := os.ReadFile(paths[i])
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to load trusted client certificate.")
		}
		if !certPool.AppendCertsFromPEM(trustedCert) {
			log.Fatal().Msgf("Failed to append trusted client certificate %v to certificate pool.", paths[i])
		}
	}

	config.ClientCAs = certPool
	config.ClientAuth = tls.RequireAndVerifyClientCert

	return config
}

func getTlsConfig(log *zerolog.Logger, c *AppConfig) *tls.Config {
	serverCert, err := tls.LoadX509KeyPair(c.Server.Tls.Cert, c.Server.Tls.Key)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load server certificate and key.")
	}

	trustedCert, err := os.ReadFile(c.Server.Tls.Ca)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load trusted server certificate.")
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(trustedCert) {
		log.Fatal().Msgf("Failed to append trusted server certificate %v to certificate pool.", c.Server.Tls.Ca)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		RootCAs:      certPool,
	}

	return tlsConfig
}

func getGrpcCrednetials(log *zerolog.Logger, c *AppConfig, opts *CliOpts) (grpc.ServerOption, bool) {
	mode := TlsMode(c.Server.Tls.Mode)

	switch mode {
	case NONE_TLS_MODE:
		return nil, false
	case TLS_TLS_MODE:
		return grpc.Creds(credentials.NewTLS(getTlsConfig(log, c))), true
	case MTLS_TLS_MODE:
		return grpc.Creds(credentials.NewTLS(getMtlsCofig(log, c, opts))), true
	default:
		log.Fatal().Msgf("Invalid TLS mode: %v. It must be one of: none, tls, mtls.", mode)
	}

	return nil, false
}

func appConfigToDaemonsConfig(c *AppConfig) *dto.DaemonsConfig {
	acdTodc := func(c *AppConfigDaemon) *dto.DaemonConfig {
		return &dto.DaemonConfig{
			Url:  c.Url,
			User: c.User,
			Pass: c.Pass,
		}
	}

	return &dto.DaemonsConfig{
		Xmr: dto.XMRDaemonConfig(*acdTodc(&c.Coin.Xmr.Daemon)),
		Btc: dto.BTCDaemonConfig(*acdTodc(&c.Coin.Btc.Daemon)),
		Ltc: dto.LTCDaemonConfig(*acdTodc(&c.Coin.Ltc.Daemon)),
		Eth: dto.ETHDaemonConfig(*acdTodc(&c.Coin.Eth.Daemon)),
		Bnb: dto.BNBDaemonConfig(*acdTodc(&c.Coin.Bnb.Daemon)),
	}
}

func getLogger() *zerolog.Logger {
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Caller().Logger()
	return &logger
}

func NewApp(opts CliOpts) *App {
	ctx, cancel := context.WithCancel(context.Background())
	log := getLogger()

	conf, err := NewAppConfig(opts.ConfigPath)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	dbUrl := fmt.Sprintf("postgresql://%v:%v@%v:%v/%v", conf.Database.User, conf.Database.Pass, conf.Database.Host, conf.Database.Port, conf.Database.Name)
	connPool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	pp, err := processor.NewPaymentProcessor(ctx, connPool, appConfigToDaemonsConfig(conf), log)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	return &App{
		log:              log,
		ctxCancel:        cancel,
		opts:             &opts,
		config:           conf,
		dbConnPool:       connPool,
		paymentProcessor: pp,
	}
}
