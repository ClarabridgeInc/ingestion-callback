package cmd

import (
	"ingestion-callback/internal/health"
	"net/http"

	"github.com/spf13/cobra"
)

func init() {
	serverCmd.PersistentFlags().StringVar(&serverCmdFlags.port, "port", ":8000", "server port")
	rootCmd.AddCommand(serverCmd)
}

var (
	serverCmdFlags struct {
		port string
	}
	serverCmd = &cobra.Command{
		Use:           "server",
		Short:         "Run go-services-seed server",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			log := global.logger.Named("go-services.seed")

			// create http multiplexer
			mux := http.NewServeMux()

			log.Debug(" > setting up health API...")
			mux.Handle("/health", &health.API{Logger: log.Named("api.health")})

			log.Debug(" >http handlers registered, starting server...")
<<<<<<< HEAD
			rootContext := cmd.Context()
			cfg, err := config.LoadDefaultConfig(rootContext)
			if err != nil {
				return err
			}

			consumer, err := sqsconsumer.NewConsumer(
				rootContext, sqsconsumer.Config{
					Logger:          log.Named("sqsconsumer.consumer"),
					Queue:           sqsconsumer.Queue{Name: global.cfg.SQS.Name},
					ReceiverDeleter: sqs.NewFromConfig(cfg),
					S3Reader: storage.S3Reader{
						Bucket: global.cfg.S3.Bucket,
						Reader: s3.NewFromConfig(cfg),
					},
					Executor: callback.NewCallbackExecutor(
						callback.Config{
							Timeout: 30 * time.Second,
						},
					),
				},
			)
			consumer.Consume(rootContext)
=======
>>>>>>> main
			return http.ListenAndServe(serverCmdFlags.port, mux)
		},
	}
)
