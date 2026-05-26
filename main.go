package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var version string

func loadConfig(path string) (*AnykConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}
	cfg := &AnykConfig{}
	if err := yaml.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("cannot unmarshal config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, nil
}

func runCmd() *cobra.Command {
	var configPath *string
	var periodic *time.Duration

	cmd := &cobra.Command{
		Use:   "run",
		Short: "run one health check cycle and update BGP routes",
		RunE: func(cmd *cobra.Command, args []string) error {
			if *periodic == 0 {
				cfg, err := loadConfig(*configPath)
				if err != nil {
					return err
				}
				return Run(cmd.Context(), cfg.ASN, cfg.Services)
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			log := WithLogger(ctx)

			cycle := func() {
				cfg, err := loadConfig(*configPath)
				if err != nil {
					log.Error().Err(err).Msg("failed to load config")
					return
				}
				if err := Run(ctx, cfg.ASN, cfg.Services); err != nil {
					log.Error().Err(err).Msg("cycle failed")
				}
			}

			cycle()

			ticker := time.NewTicker(*periodic)
			defer ticker.Stop()

			sigusr := make(chan os.Signal, 1)
			signal.Notify(sigusr, syscall.SIGUSR1)
			defer signal.Stop(sigusr)

			for {
				select {
				case <-ctx.Done():
					log.Info().Msg("shutting down")
					return nil
				case <-sigusr:
					cycle()
					ticker.Reset(*periodic)
				case <-ticker.C:
					cycle()
				}
			}
		},
	}

	configPath = cmd.PersistentFlags().StringP("config", "c", "/etc/anyk.yml", "path to an anyk config file")
	periodic = cmd.Flags().Duration("periodic", 0, "run as daemon, repeating every interval (e.g. 5s, 1m); 0 runs once")
	return cmd
}

func cleanupCmd() *cobra.Command {
	var configPath *string

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "withdraw all BGP announcements and remove all static routes",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(*configPath)
			if err != nil {
				return err
			}
			for i := range cfg.Services {
				cfg.Services[i].Active = false
			}
			return Run(cmd.Context(), cfg.ASN, cfg.Services)
		},
	}

	configPath = cmd.PersistentFlags().StringP("config", "c", "/etc/anyk.yml", "path to an anyk config file")
	return cmd
}

type contextKey string

const logKey contextKey = "log"

func WithLogger(ctx context.Context) zerolog.Logger {
	return ctx.Value(logKey).(zerolog.Logger)
}

func main() {
	var logLevel string

	l := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"}).
		With().Timestamp().Logger()

	root := &cobra.Command{
		Use:           "anyk",
		Short:         "Anycast healthcheck & routing",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			lvl, err := zerolog.ParseLevel(logLevel)
			if err != nil {
				return fmt.Errorf("invalid log level %q", logLevel)
			}

			zerolog.SetGlobalLevel(lvl)
			return nil
		},
	}

	root.AddCommand(runCmd(), cleanupCmd())
	root.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")

	if err := root.ExecuteContext(context.WithValue(context.Background(), logKey, l)); err != nil {
		l.Fatal().Err(err).Msg("anycast loop failed")
	}
}
