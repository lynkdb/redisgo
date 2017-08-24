// Copyright 2014 Eryx <evorui аt gmаil dοt cοm>, All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package redisgo // import "code.hooto.com/lynkdb/redisgo"

import (
	"fmt"
	"net"
	"runtime"
	"time"
)

type Connector struct {
	clients chan *client
	cfg     Config
	copts   *connOptions
}

type connOptions struct {
	net     string
	addr    string
	timeout time.Duration
	auth    string
}

func NewConnector(cfg Config) (*Connector, error) {

	if cfg.MaxConn < 1 {
		cfg.MaxConn = 1
	} else {
		maxconn := runtime.NumCPU()
		if maxconn > 10 {
			maxconn = 10
		}
		if cfg.MaxConn > maxconn {
			cfg.MaxConn = maxconn
		}
	}

	copts := &connOptions{
		timeout: time.Duration(cfg.Timeout) * time.Second,
		auth:    cfg.Auth,
	}

	if copts.timeout < (1 * time.Second) {
		copts.timeout = 1 * time.Second
	} else if copts.timeout > (600 * time.Second) {
		copts.timeout = 600 * time.Second
	}

	if len(cfg.Socket) > 2 {
		if _, err := net.ResolveUnixAddr("unix", cfg.Socket); err == nil {
			copts.net, copts.addr = "unix", cfg.Socket
		}
	}

	if copts.net == "" {
		copts.addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		if _, err := net.ResolveTCPAddr("tcp", copts.addr); err != nil {
			return nil, err
		}
		copts.net = "tcp"
	}

	c := &Connector{
		clients: make(chan *client, cfg.MaxConn),
		cfg:     cfg,
		copts:   copts,
	}

	for i := 0; i < cfg.MaxConn; i++ {
		cli, err := newClient(c.copts)
		if err != nil {
			return c, err
		}
		c.clients <- cli
	}

	return c, nil
}

func (c *Connector) Cmd(cmd string, args ...interface{}) *Result {
	cli, _ := c.pull()
	defer c.push(cli)

	return cli.Cmd(cmd, args...)
}

func (c *Connector) Close() {
	for i := 0; i < c.cfg.MaxConn; i++ {
		cli, _ := c.pull()
		cli.Close()
	}
}

func (c *Connector) push(cli *client) {
	c.clients <- cli
}

func (c *Connector) pull() (cli *client, err error) {
	return <-c.clients, nil
}
