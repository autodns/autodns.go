// Copyright 2025 Jelly Terra <jellyterra@symboltics.com>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

import "encoding/json"

func MarshalJSON[T any](v T) []byte {
	data, _ := json.Marshal(v)
	return data
}

func UnmarshalJSON[T any](data []byte, v T) (T, error) {
	return v, json.Unmarshal(data, v)
}
