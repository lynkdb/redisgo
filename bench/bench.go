// Copyright 2020 Eryx <evorui аt gmаil dοt cοm>, All rights reserved.
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

package main

import (
	"fmt"
	"time"

	"github.com/hooto/hflag4g/hflag"
	kvbench "github.com/lynkdb/lynkbench/kvbench/v1"
	"github.com/lynkdb/redisgo"
)

var (
	host = "127.0.0.1"
	port = 6379
	auth = ""
	err  error
)

func main() {

	mode := hflag.Value("mode").String()

	switch mode {

	case "chart":
		if err := kvbench.ChartOutput(); err != nil {
			panic(err)
		}

	case "node-x1":

		if v, ok := hflag.ValueOK("redis_host"); ok {
			host = v.String()
		}

		if v, ok := hflag.ValueOK("redis_port"); ok {
			port = v.Int()
		}

		if err := benchNodeAction(1); err != nil {
			panic(err)
		}

	default:
		fmt.Println("invalid mode")
	}
}

type benchNode struct {
	db      *redisgo.Connector
	nodeNum int
}

func (it *benchNode) Attrs() []string {
	return []string{
		fmt.Sprintf("node-x%d", it.nodeNum),
	}
}

func (it *benchNode) Write(k, v []byte) kvbench.ResultStatus {
	if rs := it.db.Cmd("set", k, v); rs.OK() {
		return kvbench.ResultOK
	}
	return kvbench.ResultERR
}

func (it *benchNode) Read(k []byte) kvbench.ResultStatus {
	if rs := it.db.Cmd("get", k); rs.OK() {
		return kvbench.ResultOK
	}
	return kvbench.ResultERR
}

func (it *benchNode) Clean() error {

	if it.db != nil {
		it.db.Cmd("FLUSHALL")
		it.db.Close()
		time.Sleep(60e9)
	}

	it.db, err = redisgo.NewConnector(redisgo.Config{
		Host:    host,
		Port:    uint16(port),
		Timeout: 3,
		MaxConn: 1,
		Auth:    auth,
	})
	if err != nil {
		return err
	}

	return err
}

func benchNodeAction(n int) error {

	kvBench, err := kvbench.NewKeyValueBench()
	if err != nil {
		return err
	}

	bc := &benchNode{
		nodeNum: n,
	}

	return kvBench.Run(bc)
}
