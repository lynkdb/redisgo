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

package main

import (
	"fmt"

	"github.com/lynkdb/redisgo"
)

func main() {

	fmt.Println("Connect")
	conn, err := redisgo.NewConnector(redisgo.Config{
		Host:    "127.0.0.1",
		Port:    6379,
		Timeout: 3, // timeout in second, default to 10
		MaxConn: 1, // max connection number, default to 1
		// Auth:    "foobared",
	})
	if err != nil {
		print_err(err.Error())
		return
	} else {
		print_ok("OK")
	}
	defer conn.Close()

	{
		fmt.Println("SET API::Bool() bool")
		conn.Cmd("set", "true", "True")
		if rs := conn.Cmd("get", "true"); rs.OK() && rs.Bool() {
			print_ok("OK")
		} else {
			print_err("Failed " + rs.String())
		}
	}

	{
		fmt.Println("SET API::String() string")
		conn.Cmd("set", "aa", "val-aaaaaaaaaaaaaaaaaa")
		conn.Cmd("multi_set", []string{
			"bb", "val-bbbbbbbbbbbbbbbbbb",
			"cc", "val-cccccccccccccccccc",
		})
		if rs := conn.Cmd("get", "aa"); rs.OK() {
			print_ok("OK (get by string) " + rs.String())
		} else {
			print_ok("ER " + rs.String())
		}
		if rs := conn.Cmd("get", []byte("aa")); rs.OK() {
			print_ok("OK (get by bytes) " + rs.String())
		} else {
			print_ok("ER " + rs.String())
		}
	}

	{
		fmt.Println("API::List() MGET")
		if rs := conn.Cmd("mget", "aa", "bb"); rs.OK() {
			print_ok(fmt.Sprintf("OK len: %d", len(rs.List())))
			for i, v := range rs.List() {
				print_ok(fmt.Sprintf("  No. %d value:%s", i, v.String()))
			}
		} else {
			print_err("ER " + rs.String())
		}
	}

	{
		fmt.Println("API::List() MGET bytes")
		bkeys := []interface{}{[]byte("aa"), []byte("bb")}
		if rs := conn.Cmd("mget", bkeys...); rs.OK() {
			print_ok(fmt.Sprintf("OK len: %d", len(rs.List())))
			for i, v := range rs.List() {
				print_ok(fmt.Sprintf("  No. %d value:%s", i, v.String()))
			}
		} else {
			print_err("ER " + rs.String())
		}
	}

	{
		fmt.Println("SCAN")
		if rs := conn.Cmd("scan", 0, "COUNT", 2); rs.OK() && len(rs.Items) == 2 {
			print_ok(fmt.Sprintf("OK multi len: %d", len(rs.Items)))
			if len(rs.Items[1].Items) == 2 {
				print_ok(fmt.Sprintf("  offset %s", rs.Items[0].String()))
				for i, v := range rs.Items[1].Items {
					print_ok(fmt.Sprintf("  No. %d key:%s", i, v.String()))
				}
			} else {
				print_ok(fmt.Sprintf("  ERR items len: %d", len(rs.Items[0].Items)))
			}
		} else {
			print_err("ER " + rs.String())
		}
	}

	{
		fmt.Println("ZADD")
		conn.Cmd("zadd", "z", 3, "a")
		conn.Cmd("zadd", "z", -2, "b")
		conn.Cmd("zadd", "z", 5, "c")
		if rs := conn.Cmd("zscan", "z", 0, "COUNT", 3); rs.OK() && len(rs.Items) == 2 {
			print_ok(fmt.Sprintf("OK multi len: %d", len(rs.Items)))
			if rs.Items[1].KvLen() == 3 {
				rs.Items[1].KvEach(func(k, v *redisgo.Result) {
					print_ok(fmt.Sprintf("  key:%s value:%s", k.String(), v.String()))
				})
			} else {
				print_ok(fmt.Sprintf("  ERR items len: %d", rs.Items[0].KvLen()))
			}
		} else {
			print_err("ER " + rs.String())
		}
	}

	{
		fmt.Println("SET + INCRBY")
		conn.Cmd("set", "key", 10)
		if rs := conn.Cmd("incrby", "key", 1).Int(); rs == 11 {
			print_ok("OK")
		} else {
			print_err("ERR")
		}
	}

	{
		fmt.Println("SET EX, TTL")
		conn.Cmd("set", "key", 123456, "EX", 300)
		if rs := conn.Cmd("ttl", "key").Int(); rs > 298 {
			print_ok("OK")
		} else {
			print_err("ERR")
		}
	}

	{
		fmt.Println("HASH M SET")
		if rs := conn.Cmd("hmset", "zone", "c1", "v-01", "c2", "v-02"); rs.OK() {
			print_ok("OK")
		} else {
			print_err("ERR")
		}

		fmt.Println("HASH M GET")
		if rs := conn.Cmd("hmget", "zone", "c1", "c2"); rs.OK() && len(rs.Items) == 2 {
			ls := rs.List()
			for i, v := range ls {
				print_ok(fmt.Sprintf("No. %d value:%s", i, v.String()))
			}
		} else {
			print_err("ER " + rs.String())
		}

		fmt.Println("HASH GET ALL")
		if rs := conn.Cmd("hgetall", "zone"); rs.OK() && rs.KvLen() == 2 {
			print_ok(fmt.Sprintf("OK multi len: %d", rs.KvLen()))
			rs.KvEach(func(k, v *redisgo.Result) {
				print_ok(fmt.Sprintf("  key:%s value:%s", k.String(), v.String()))
			})
		} else {
			print_err("ER " + rs.String())
		}
	}

	{
		fmt.Println("SET float")
		conn.Cmd("set", "float", 123.456)
		if rs := conn.Cmd("get", "float").Float64(); rs == 123.456 {
			print_ok("OK")
		} else {
			print_err("ER")
		}
	}

	{
		fmt.Println("API::List()")
		conn.Cmd("lpush", "queue", "q-1111111111111")
		conn.Cmd("lpush", "queue", "q-2222222222222", "q-3333333333333")
		if rs := conn.Cmd("lrange", "queue", 0, -1); rs.OK() && len(rs.Items) >= 3 {
			print_ok("OK LRANGE")
			for i, v := range rs.Items {
				print_ok(fmt.Sprintf("  No. %d value:%s", i, v.String()))
			}
		} else {
			print_err("ER")
		}

		for {
			rs := conn.Cmd("lpop", "queue")
			if !rs.OK() {
				break
			}
			print_ok("OK LPOP " + rs.String())
		}
	}

	{
		fmt.Println("SET JSON")
		conn.Cmd("set", "json_key", "{\"name\": \"test obj.name\", \"value\": \"test obj.value\"}")
		if rs := conn.Cmd("get", "json_key"); rs.OK() {
			var rs_obj struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			}
			if err := rs.JsonDecode(&rs_obj); err == nil {
				print_ok(fmt.Sprintf("OK key:%s value:%s", rs_obj.Name, rs_obj.Value))
			} else {
				print_err("ER " + err.Error())
			}
		} else {
			print_err("ER " + rs.String())
		}
	}
}

func print_ok(msg string) {
	fmt.Printf("\033[32m  %s \033[0m\n", msg)
}

func print_err(msg string) {
	fmt.Printf("\033[31m  %s \033[0m\n", msg)
}
