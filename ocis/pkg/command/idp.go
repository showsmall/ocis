package command

import (
	"github.com/micro/cli/v2"
	"github.com/owncloud/ocis/idp/pkg/command"
	svcconfig "github.com/owncloud/ocis/idp/pkg/config"
	"github.com/owncloud/ocis/idp/pkg/flagset"
	"github.com/owncloud/ocis/ocis/pkg/config"
	"github.com/owncloud/ocis/ocis/pkg/register"
	"github.com/owncloud/ocis/ocis/pkg/version"
)

// IDPCommand is the entrypoint for the idp command.
func IDPCommand(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "idp",
		Usage:    "Start idp server",
		Category: "Extensions",
		Flags:    flagset.ServerWithConfig(cfg.IDP),
		Subcommands: []*cli.Command{
			command.PrintVersion(cfg.IDP),
		},
		Action: func(c *cli.Context) error {
			idpCommand := command.Server(configureIDP(cfg))

			if err := idpCommand.Before(c); err != nil {
				return err
			}

			return cli.HandleAction(idpCommand.Action, c)
		},
	}
}

// TODO tracing config should be defined on the top level and cascade down to subcommands to avoid functions like this one
func configureIDP(cfg *config.Config) *svcconfig.Config {
	cfg.IDP.Log.Level = cfg.Log.Level
	cfg.IDP.Log.Pretty = cfg.Log.Pretty
	cfg.IDP.Log.Color = cfg.Log.Color
	cfg.IDP.HTTP.TLS = false
	cfg.IDP.Service.Version = version.String

	if cfg.Tracing.Enabled {
		cfg.IDP.Tracing.Enabled = cfg.Tracing.Enabled
		cfg.IDP.Tracing.Type = cfg.Tracing.Type
		cfg.IDP.Tracing.Endpoint = cfg.Tracing.Endpoint
		cfg.IDP.Tracing.Collector = cfg.Tracing.Collector
	}

	return cfg.IDP
}

func init() {
	register.AddCommand(IDPCommand)
}