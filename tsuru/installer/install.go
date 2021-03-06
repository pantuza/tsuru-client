// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package installer

import (
	"fmt"
	"strings"

	"github.com/tsuru/config"
	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
)

type Install struct {
	fs     *gnuflag.FlagSet
	config string
}

func (c *Install) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "install",
		Usage:   "install",
		Desc:    "",
		MinArgs: 0,
	}
}

func (c *Install) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("install", gnuflag.ExitOnError)
		c.fs.StringVar(&c.config, "c", "install.yml", "Configuration file")
		c.fs.StringVar(&c.config, "config", "install.yml", "Configuration file")
	}
	return c.fs
}

func (c *Install) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	config, err := parseConfigFile(c.config)
	if err != nil {
		return err
	}
	i, err := NewDockerMachine(config)
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to create machine: %s\n", err)
		return err
	}
	m, err := i.CreateMachine()
	if err != nil {
		fmt.Fprintf(context.Stderr, "Error creating machine: %s\n", err)
		return err
	}
	fmt.Fprintf(context.Stdout, "Machine %s successfully created!\n", m.IP)
	err = i.CreateRegistryCertificate()
	if err != nil {
		return err
	}
	fmt.Fprint(context.Stdout, "Registry certificates successfully created!\n")
	installConfig := NewInstallConfig()
	for _, component := range TsuruComponents {
		fmt.Fprintf(context.Stdout, "Installing %s\n", component.Name())
		err := component.Install(m, installConfig)
		if err != nil {
			fmt.Fprintf(context.Stderr, "Error Installing %s: %s\n", component.Name(), err)
			return err
		}
		fmt.Fprintf(context.Stdout, "%s successfully installed!\n", component.Name())
	}
	fmt.Fprint(context.Stdout, c.buildStatusTable(TsuruComponents, m).String())
	return nil
}

func (c *Install) buildStatusTable(components []TsuruComponent, m *Machine) *cmd.Table {
	t := cmd.NewTable()
	t.Headers = cmd.Row{"Component", "Address", "State"}
	t.LineSeparator = true
	for _, component := range components {
		status, err := component.Status(m)
		if err != nil {
			t.AddRow(cmd.Row{component.Name(), "", fmt.Sprintf("%s", err)})
			continue
		}
		addresses := strings.Join(status.addresses, "\n")
		t.AddRow(cmd.Row{component.Name(), addresses, status.containerState.StateString()})
	}
	return t
}

type Uninstall struct {
	fs     *gnuflag.FlagSet
	config string
}

func (c *Uninstall) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "uninstall",
		Usage:   "uninstall",
		Desc:    "",
		MinArgs: 0,
	}
}

func (c *Uninstall) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("uninstall", gnuflag.ExitOnError)
		c.fs.StringVar(&c.config, "c", "install.yml", "Configuration file")
		c.fs.StringVar(&c.config, "config", "install.yml", "Configuration file")
	}
	return c.fs
}

func (c *Uninstall) Run(context *cmd.Context, client *cmd.Client) error {
	config, err := parseConfigFile(c.config)
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to read configuration file: %s\n", err)
		return err
	}
	d, err := NewDockerMachine(config)
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to delete machine: %s\n", err)
		return err
	}
	err = d.DeleteMachine(&Machine{})
	if err != nil {
		fmt.Fprintf(context.Stderr, "Failed to delete machine: %s\n", err)
		return err
	}
	fmt.Fprintln(context.Stdout, "Machine successfully removed!")
	return nil
}

func parseConfigFile(file string) (*DockerMachineConfig, error) {
	err := config.ReadConfigFile(file)
	if err != nil {
		return nil, err
	}
	driverName, err := config.GetString("driver:name")
	if err != nil {
		return nil, err
	}
	driverOpts := make(map[string]interface{})
	opts, _ := config.Get("driver:options")
	if opts != nil {
		for k, v := range opts.(map[interface{}]interface{}) {
			switch k := k.(type) {
			case string:
				driverOpts[k] = v
			}
		}
	}
	return &DockerMachineConfig{
		DriverName: driverName,
		DriverOpts: driverOpts,
	}, nil
}
