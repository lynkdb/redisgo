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

package redisgo // import "github.com/lynkdb/redisgo"

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"strconv"
	"time"
)

var (
	bufio_size = 4096
	delim      = []byte{'\r', '\n'}
	err_parse  = errors.New("parse error")
	err_auth   = errors.New("auth failed")
)

type client struct {
	sock   net.Conn
	reader *bufio.Reader
	copts  *connOptions
}

func newClient(copts *connOptions) (*client, error) {

	sock, err := net.Dial(copts.net, copts.addr)
	if err != nil {
		return nil, err
	}

	cli := &client{
		sock:   sock,
		reader: bufio.NewReaderSize(sock, bufio_size),
		copts:  copts,
	}

	if copts.auth != "" {
		if rs := cli.Cmd("auth", copts.auth); !rs.OK() {
			return nil, err_auth
		}
	}

	return cli, nil
}

func (c *client) Cmd(cmd string, args ...interface{}) *Result {

	buf, err := send_buf_cmd(cmd, args)
	if err != nil {
		return newResult(ResultBadArgument, err)
	}

	if c.sock == nil {
		sock, err := net.Dial(c.copts.net, c.copts.addr)
		if err != nil {
			return newResult(ResultNetworkException, err)
		}
		c.sock = sock
		c.reader = bufio.NewReaderSize(sock, bufio_size)

		if c.copts.auth != "" {
			if rs := c.Cmd("auth", c.copts.auth); !rs.OK() {
				return newResult(ResultNoAuth, err_auth)
			}
		}
	}

	c.sock.SetDeadline(time.Now().Add(c.copts.timeout))
	if _, err = c.sock.Write(buf); err != nil {
		return newResult(ResultNetworkException, err)
	}

	rs, err := c.cmd_parse()
	if err != nil {
		if ev, ok := err.(*net.OpError); ok && ev.Timeout() {
			return newResult(ResultTimeout, err)
		}
		return newResult(ResultNetworkException, err)
	}

	return rs
}

func cmd_parse_read(reader *bufio.Reader, size int) ([]byte, error) {
	bs := make([]byte, size)
	ni := 0
	for {
		n, err := reader.Read(bs[ni:])
		if err != nil {
			return nil, err
		}
		if (ni + n) >= size {
			break
		}
		ni += n
	}
	return bs, nil
}

func (c *client) cmd_parse() (*Result, error) {

	bs, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(bs) < 4 {
		return nil, err_parse
	}

	rs := newResult(0, nil)

	switch bs[0] {

	// Errors
	case '-':
		rs.data = bytes_clone(bs[1 : len(bs)-2])
		rs.cap = 0
		rs.Status = ResultError

	// Simple Strings, Integers
	case '+', ':':
		rs.data = bytes_clone(bs[1 : len(bs)-2])
		rs.cap = 1

	// Bulk Strings
	case '$':
		size, err := strconv.Atoi(string(bs[1 : len(bs)-2]))
		if err != nil || size < -1 {
			return nil, err_parse
		}
		if size > 0 {
			bs2, err := cmd_parse_read(c.reader, size+2)
			if err != nil {
				return nil, err
			}
			rs.data = bytes_clone(bs2[:len(bs2)-2])
			rs.cap = 1
		} else {
			rs.cap = 0
		}

	// Array
	case '*':
		size, err := strconv.Atoi(string(bs[1 : len(bs)-2]))
		if err != nil || size < -1 {
			return nil, err_parse
		}

		rs.cap = size
		if err = cmd_parse_array(rs, c.reader); err != nil {
			return nil, err
		}

	// protocol error
	default:
		return nil, err_parse
	}

	if rs.Status == 0 {
		if rs.cap == 0 || (len(rs.data) == 0 && len(rs.Items) == 0) {
			rs.Status = ResultNotFound
		} else if (rs.cap == 1 && len(rs.data) > 0) || len(rs.Items) >= rs.cap {
			rs.Status = ResultOK
		} else {
			rs.Status = ResultUnknown
		}
	}

	return rs, nil
}

func cmd_parse_array(rs *Result, reader *bufio.Reader) error {

	for i := 0; i < rs.cap; i++ {

		bs, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}
		if len(bs) < 4 {
			return err_parse
		}

		switch bs[0] {

		// Bulk Strings
		case '$':
			size, err := strconv.Atoi(string(bs[1 : len(bs)-2]))
			if err != nil || size < -1 {
				return err_parse
			}
			if size > 0 {
				bs2, err := cmd_parse_read(reader, size+2)
				if err != nil {
					return err
				}
				rs.Items = append(rs.Items, &Result{
					data: bytes_clone(bs2[:(len(bs2) - 2)]),
					cap:  1,
				})
			} else {
				rs.Items = append(rs.Items, &Result{
					cap: 0,
				})
			}

		// Array
		case '*':
			size, err := strconv.Atoi(string(bs[1 : len(bs)-2]))
			if err != nil || size < -1 {
				return err_parse
			}

			rs2 := &Result{cap: size}
			if size > 0 {
				if err := cmd_parse_array(rs2, reader); err != nil {
					return err
				}
			}
			rs.Items = append(rs.Items, rs2)

		// protocol error
		default:
			return err_parse
		}
	}

	return nil
}

func (c *client) Close() error {
	if c.sock != nil {
		c.sock.Close()
		c.sock = nil
	}
	return nil
}

func send_buf_cmd(cmd string, args []interface{}) ([]byte, error) {

	var buf bytes.Buffer

	buf.WriteByte('*')
	buf.Write([]byte(strconv.Itoa(len(args) + 1)))
	buf.Write(delim)

	send_buf_ss(&buf, &cmd)

	for _, arg := range args {

		var s string

		switch argt := arg.(type) {

		case []byte:
			send_buf_bs(&buf, argt)
			continue

		case string:
			s = argt

		case int:
			s = strconv.FormatInt(int64(argt), 10)

		case int8:
			s = strconv.FormatInt(int64(argt), 10)

		case int16:
			s = strconv.FormatInt(int64(argt), 10)

		case int32:
			s = strconv.FormatInt(int64(argt), 10)

		case int64:
			s = strconv.FormatInt(argt, 10)

		case uint:
			s = strconv.FormatUint(uint64(argt), 10)

		case uint8:
			s = strconv.FormatUint(uint64(argt), 10)

		case uint16:
			s = strconv.FormatUint(uint64(argt), 10)

		case uint32:
			s = strconv.FormatUint(uint64(argt), 10)

		case uint64:
			s = strconv.FormatUint(argt, 10)

		case float32:
			s = strconv.FormatFloat(float64(argt), 'f', -1, 32)

		case float64:
			s = strconv.FormatFloat(argt, 'f', -1, 64)

		case bool:
			if argt {
				s = "1"
			} else {
				s = "0"
			}

		case nil:
			s = ""

		default:
			return []byte{}, errors.New("bad arguments")
		}

		send_buf_ss(&buf, &s)
	}

	return buf.Bytes(), nil
}

func send_buf_bs(buf *bytes.Buffer, data []byte) {
	buf.WriteByte('$')
	buf.WriteString(strconv.FormatInt(int64(len(data)), 10))
	buf.Write(delim)
	buf.Write(data)
	buf.Write(delim)
}

func send_buf_ss(buf *bytes.Buffer, data *string) {
	buf.WriteByte('$')
	buf.WriteString(strconv.FormatInt(int64(len(*data)), 10))
	buf.Write(delim)
	buf.WriteString(*data)
	buf.Write(delim)
}

func bytes_clone(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}
