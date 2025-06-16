// Copyright 2025 Jelly Terra <jellyterra@proton.me>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package cloudflare

import (
	"context"
	"fmt"
	"github.com/autodns/autodns.go/core"
	"github.com/cloudflare/cloudflare-go"
)

type Registry struct {
	Ctx context.Context
	API *cloudflare.API
	RC  *cloudflare.ResourceContainer

	RecordMap map[string][]cloudflare.DNSRecord
}

func (r *Registry) AppendRecord(record *core.Record) error {
	_, err := r.API.CreateDNSRecord(r.Ctx, r.RC, cloudflare.CreateDNSRecordParams{
		Type:    record.Type,
		Name:    record.CanonicalName,
		Content: record.Value,
		TTL:     record.TTL,
	})
	return err
}

func (r *Registry) DeleteRecord(record *core.Record) error {
	for _, rec := range r.RecordMap[record.CanonicalName] {
		if rec.Content == record.Value {
			err := r.API.DeleteDNSRecord(r.Ctx, r.RC, rec.ID)
			if err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func (r *Registry) DeleteAllRecordsWithDomain(domain string) error {
	for _, record := range r.RecordMap[domain] {
		err := r.API.DeleteDNSRecord(r.Ctx, r.RC, record.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) Close() error { return nil }

func Build(config map[string]string) (core.Registry, error) {
	var (
		apiToken = config["api_token"]
		zone     = config["zone"]
	)
	if apiToken == "" || zone == "" {
		return nil, fmt.Errorf("cloudflare: require [api_token, zone]")
	}

	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, err
	}

	zoneId, err := api.ZoneIDByName(zone)
	if err != nil {
		return nil, err
	}

	r := &Registry{
		Ctx:       context.Background(),
		API:       api,
		RC:        cloudflare.ZoneIdentifier(zoneId),
		RecordMap: map[string][]cloudflare.DNSRecord{},
	}

	records, _, err := r.API.ListDNSRecords(r.Ctx, r.RC, cloudflare.ListDNSRecordsParams{})
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		r.RecordMap[record.Name] = append(r.RecordMap[record.Name], record)
	}

	return r, nil
}

func init() {
	core.RegistryBuilders["cloudflare"] = Build
}
