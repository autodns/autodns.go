// Copyright 2025 Jelly Terra <jellyterra@symboltics.com>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package core

import (
	"context"
	redis "github.com/redis/rueidis"
	"maps"
)

type Database struct {
	redis.Client
}

const (
	PREFIX_ROLE_TOKEN          = "RoleToken:"
	PREFIX_ROLE_SUBDOMAIN_GLOB = "RoleSubdomainGlob:"
	PREFIX_REGISTRY            = "Registry:"

	KEY_DOMAIN_REGISTRY = "DomainRegistry"
)

func (db *Database) UpdateRoleToken(ctx context.Context, role, token, desc string) error {
	return db.Do(ctx, db.B().Hset().Key(PREFIX_ROLE_TOKEN+role).FieldValue().FieldValue(token, desc).Build()).Error()
}

func (db *Database) MatchRoleToken(ctx context.Context, role, token string) (bool, error) {
	return db.Do(ctx, db.B().Hexists().Key(PREFIX_ROLE_TOKEN+role).Field(token).Build()).AsBool()
}

func (db *Database) DeleteRoleToken(ctx context.Context, role, token string) error {
	return db.Do(ctx, db.B().Hdel().Key(PREFIX_ROLE_TOKEN+role).Field(token).Build()).Error()
}

func (db *Database) UpdateRoleSubdomainGlob(ctx context.Context, role, domain, glob string) error {
	return db.Do(ctx, db.B().Hset().Key(PREFIX_ROLE_SUBDOMAIN_GLOB+role).FieldValue().FieldValue(domain, glob).Build()).Error()
}

func (db *Database) QueryRoleSubdomainGlob(ctx context.Context, role, domain string) (string, bool, error) {
	v, err := db.Do(ctx, db.B().Hget().Key(PREFIX_ROLE_SUBDOMAIN_GLOB+role).Field(domain).Build()).ToString()
	switch {
	case err == nil:
		return v, true, nil
	case redis.IsRedisNil(err):
		return "", false, nil
	default:
		return "", false, err
	}
}

func (db *Database) DeleteRoleSubdomainGlob(ctx context.Context, role, domain string) error {
	return db.Do(ctx, db.B().Hdel().Key(PREFIX_ROLE_SUBDOMAIN_GLOB+role).Field(domain).Build()).Error()
}

func (db *Database) DeleteRole(ctx context.Context, role string) error {
	err := db.Do(ctx, db.B().Del().Key(PREFIX_ROLE_TOKEN+role).Build()).Error()
	if err != nil {
		return err
	}
	return db.Do(ctx, db.B().Del().Key(PREFIX_ROLE_SUBDOMAIN_GLOB).Build()).Error()
}

func (db *Database) UpdateDomainRegistry(ctx context.Context, domain, registry string) error {
	return db.Do(ctx, db.B().Hset().Key(KEY_DOMAIN_REGISTRY).FieldValue().FieldValue(domain, registry).Build()).Error()
}

func (db *Database) QueryDomainRegistry(ctx context.Context, domain string) (string, bool, error) {
	v, err := db.Do(ctx, db.B().Hget().Key(KEY_DOMAIN_REGISTRY).Field(domain).Build()).ToString()
	switch {
	case err == nil:
		return v, true, nil
	case redis.IsRedisNil(err):
		return "", false, nil
	default:
		return "", false, err
	}
}

func (db *Database) DeleteDomainRegistry(ctx context.Context, domain string) error {
	return db.Do(ctx, db.B().Hdel().Key(KEY_DOMAIN_REGISTRY).Field(domain).Build()).Error()
}

func (db *Database) UpdateRegistryConfig(ctx context.Context, registry string, config map[string]string) error {
	return db.Do(ctx, db.B().Hset().Key(PREFIX_REGISTRY+registry).FieldValue().FieldValueIter(maps.All(config)).Build()).Error()
}

func (db *Database) QueryRegistryConfig(ctx context.Context, registry string) (map[string]string, error) {
	return db.Do(ctx, db.B().Hgetall().Key(PREFIX_REGISTRY+registry).Build()).AsStrMap()
}

func (db *Database) DeleteRegistry(ctx context.Context, registry string) error {
	return db.Do(ctx, db.B().Del().Key(PREFIX_REGISTRY+registry).Build()).Error()
}
