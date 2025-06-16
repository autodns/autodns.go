// Copyright 2025 Jelly Terra <jellyterra@proton.me>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/json"
	"io"
	"os"
)

func MarshalJSON[T any](v T) []byte {
	data, _ := json.Marshal(v)
	return data
}

func MarshalJSONToWriter(w io.Writer, v any) error {
	data, _ := json.Marshal(v)
	_, err := w.Write(data)
	return err
}

func MarshalJSONToPath(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return MarshalJSONToWriter(f, v)
}

func UnmarshalJSON[T any](data []byte, v *T) (*T, error) {
	return v, json.Unmarshal(data, v)
}

func UnmarshalJSONFromReader[T any](r io.Reader, v *T) (*T, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return UnmarshalJSON[T](data, v)
}

func UnmarshalJSONFromPath[T any](path string, v *T) (*T, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return UnmarshalJSON[T](b, v)
}
