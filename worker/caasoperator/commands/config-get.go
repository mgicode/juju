// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"fmt"

	"github.com/juju/cmd"
	"github.com/juju/gnuflag"
)

// ConfigGetCommand implements the config-get command.
type ConfigGetCommand struct {
	cmd.CommandBase
	ctx Context
	Key string // The key to show. If empty, show all.
	All bool
	out cmd.Output
}

// NewContainerspecSetCommand makes a config-get command.
func NewConfigGetCommand(ctx Context) (cmd.Command, error) {
	return &ConfigGetCommand{ctx: ctx}, nil
}

func (c *ConfigGetCommand) Info() *cmd.Info {
	doc := `
When no <key> is supplied, all keys with values or defaults are printed. If
--all is set, all known keys are printed; those without defaults or values are
reported as null. <key> and --all are mutually exclusive.
`
	return &cmd.Info{
		Name:    "config-get",
		Args:    "[<key>]",
		Purpose: "print application configuration",
		Doc:     doc,
	}
}

func (c *ConfigGetCommand) SetFlags(f *gnuflag.FlagSet) {
	c.out.AddFlags(f, "smart", cmd.DefaultFormatters)
	f.BoolVar(&c.All, "a", false, "print all keys")
	f.BoolVar(&c.All, "all", false, "")
}

func (c *ConfigGetCommand) Init(args []string) error {
	if args == nil {
		return nil
	}
	c.Key = args[0]
	if c.Key != "" && c.All {
		return fmt.Errorf("cannot use argument --all together with key %q", c.Key)
	}

	return cmd.CheckEmpty(args[1:])
}

func (c *ConfigGetCommand) Run(ctx *cmd.Context) error {
	settings, err := c.ctx.CharmConfig()
	if err != nil {
		return err
	}
	var value interface{}
	if c.Key == "" {
		if !c.All {
			for k, v := range settings {
				if v == nil {
					delete(settings, k)
				}
			}
		}
		value = settings
	} else {
		value, _ = settings[c.Key]
	}
	return c.out.Write(ctx, value)
}
