package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	err := run(context.Background(), os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	cmd := &cli.Command{
		Name:  "ec_check",
		Usage: "Elastic Cloud Check Tool",
		Commands: []*cli.Command{
			{
				Name:  "downscale",
				Usage: "calculate, if downscaling of an EC deployment is feasible based on current disk consumption",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "deployment",
						Aliases:  []string{"d"},
						Usage:    "Name of the deployment in Elastic Cloud, e.g. my-deployment",
						Local:    true,
						Required: true,
					},
					&cli.StringFlag{
						Name:  "username",
						Usage: "Username used to authenticate against Elasticsearch",
						Local: true,
					},
					&cli.StringFlag{
						Name:  "password",
						Usage: "Password used to authenticate against Elasticsearch",
						Local: true,
					},
					&cli.StringFlag{
						Name:     "region",
						Aliases:  []string{"r"},
						Usage:    "Deployment region of the Elastic Cloud deployment, e.g. azure-westeurope",
						Local:    true,
						Required: true,
					},
					&cli.StringFlag{
						Name:     "profile",
						Aliases:  []string{"p"},
						Usage:    "Deployment profile Username used to authenticate against Elasticsearch, e.g. azure-general-purpose-v2",
						Local:    true,
						Required: true,
					},
					&cli.Float64Flag{
						Name:  "headroom-pct",
						Usage: "Required available headroom in percent after downscale for the downscale to be recommended",
						Value: 25.0,
						Local: true,
					},
					&cli.BoolFlag{
						Name:    "recommend-zone-change",
						Aliases: []string{"e"},
						Usage:   "With this flag provided, downscaling recommendation will also include changing the number of zones (not recommended by Elastic)",
						Value:   false,
						Local:   true,
					},
					&cli.BoolFlag{
						Name:    "exit-code",
						Aliases: []string{"e"},
						Usage:   "With this flag provided, the exit code will be set to none 0, if downscaling is recommended",
						Value:   false,
						Local:   true,
					},
				},
				Action: downscale,
			},
			{
				Name:   "regions",
				Usage:  "return list of Elastic Cloud regions",
				Action: listRegions,
			},
			{
				Name:  "profiles",
				Usage: "return list of available profiles in a given region",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "region",
						Aliases:  []string{"r"},
						Usage:    "Deployment region of the Elastic Cloud deployment, e.g. azure-westeurope",
						Local:    true,
						Required: true,
					},
				},
				Action: listProfiles,
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enable verbose output",
				Value:   false,
			},
		},
	}

	return cmd.Run(ctx, args)
}
