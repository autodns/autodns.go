// Copyright 2025 Jelly Terra <jellyterra@symboltics.com>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package core

type Record struct {
	Type          string `json:"type"`
	CanonicalName string `json:"name"`
	Value         string `json:"value"`
	TTL           int    `json:"ttl"`
}

type Registry interface {
	AppendRecord(records *Record) error
	DeleteRecord(records *Record) error
	DeleteAllRecordsWithDomain(domain string) error
	Close() error
}

type RegistryBuilder func(config map[string]string) (Registry, error)

var RegistryBuilders = map[string]RegistryBuilder{}
