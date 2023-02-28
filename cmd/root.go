package cmd

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"ingestion-callback/internal/config"
	"os"

	"github.com/spf13/cobra"
)

var (
	global struct {
		logger *zap.Logger
		cfg    *config.Config
	}
	cfgFile      string
	rootCmdFlags struct {
		devMode       bool
		configSources []string
	}
	rootCmd = &cobra.Command{
		Use:   "ingestion-callback",
		Short: "ingestion-callback command",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// create logger
			log, err := logger(rootCmdFlags.devMode)
			if err != nil {
				return err
			}

			global.logger = log

			// check if any config parameter is passed
			if len(rootCmdFlags.configSources) == 0 {
				return fmt.Errorf("no config file provided using the --config parameter")
			}

			// read config
			log.Info("reading config", zap.Strings("configPaths", rootCmdFlags.configSources))
			cfg, err := config.ReadFromPaths(rootCmdFlags.configSources...)
			if err != nil {
				return err
			}

			global.cfg = cfg

			return nil
		},
	}
)

func init() {
	rootCmd.PersistentFlags().BoolVar(&rootCmdFlags.devMode, "dev", false, "use dev profile (disabled by default)")
	rootCmd.PersistentFlags().StringArrayVar(&rootCmdFlags.configSources, "config", []string{}, "config file")
}

func logger(dev bool) (*zap.Logger, error) {
	if dev {
		logCfg := zap.NewDevelopmentConfig()
		logCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		logCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		return logCfg.Build()
	}

	return zap.NewProduction()
}

// Execute function
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
