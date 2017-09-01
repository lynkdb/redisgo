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
	"encoding/json"
	"errors"
	"strconv"
)

const (
	_ uint8 = iota
	ResultOK
	ResultError
	ResultNotFound
	ResultBadArgument
	ResultNoAuth
	ResultServerError
	ResultNetworkException
	ResultTimeout
	ResultUnknown
)

type Result struct {
	Status uint8
	data   []byte
	cap    int
	Items  []*Result
}

func newResult(status uint8, err error) *Result {

	r := &Result{
		Status: status,
	}

	if err != nil {
		if status == 0 {
			r.Status = ResultError
		}
		r.data = []byte(err.Error())
	}

	return r
}

func (r *Result) OK() bool {
	return r.Status == ResultOK
}

func (r *Result) NotFound() bool {
	return r.Status == ResultNotFound
}

//
func (r *Result) Bytes() []byte {
	return r.data
}

func (r *Result) String() string {
	return string(r.data)
}

func (r *Result) Bool() bool {
	if len(r.data) > 0 {
		if b, err := strconv.ParseBool(string(r.data)); err == nil {
			return b
		}
	}
	return false
}

func (r *Result) Int() int {
	return int(r.Int64())
}

func (r *Result) Int8() int8 {
	return int8(r.Int64())
}

func (r *Result) Int16() int16 {
	return int16(r.Int64())
}

func (r *Result) Int32() int32 {
	return int32(r.Int64())
}

func (r *Result) Int64() int64 {
	if len(r.data) > 0 {
		if i64, err := strconv.ParseInt(string(r.data), 10, 64); err == nil {
			return i64
		}
	}
	return 0
}

func (r *Result) Uint() uint {
	return uint(r.Uint64())
}

func (r *Result) Uint8() uint8 {
	return uint8(r.Uint64())
}

func (r *Result) Uint16() uint16 {
	return uint16(r.Uint64())
}

func (r *Result) Uint32() uint32 {
	return uint32(r.Uint64())
}

func (r *Result) Uint64() uint64 {
	if len(r.data) > 0 {
		if i64, err := strconv.ParseUint(string(r.data), 10, 64); err == nil {
			return i64
		}
	}
	return 0
}

func (r *Result) Float32() float32 {
	return float32(r.Float64())
}

func (r *Result) Float64() float64 {
	if len(r.data) > 0 {
		if f64, err := strconv.ParseFloat(string(r.data), 64); err == nil {
			return f64
		}
	}
	return 0
}

func (r *Result) JsonDecode(v interface{}) error {
	if len(r.data) < 2 {
		return errors.New("json: invalid format")
	}
	return json.Unmarshal(r.data, &v)
}

func (r *Result) List() []*Result {
	return r.Items
}

func (r *Result) KvLen() int {
	return len(r.Items) / 2
}

func (r *Result) KvEach(fn func(key, value *Result)) int {
	for i := 1; i < len(r.Items); i += 2 {
		fn(r.Items[i-1], r.Items[i])
	}
	return r.KvLen()
}
