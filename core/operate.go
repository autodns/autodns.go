// Copyright 2025 Jelly Terra <jellyterra@symboltics.com>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package core

import (
	"context"
	"fmt"
	"golang.org/x/net/idna"
	"regexp"
	"sync"
)

const (
	OP_UPDATE = "update"
	OP_DELETE = "delete"
)

type GlobalCache struct {
	Glob     map[string]*regexp.Regexp
	GlobLock sync.RWMutex
}

func NewGlobalCache() *GlobalCache {
	return &GlobalCache{
		Glob: make(map[string]*regexp.Regexp, 8),
	}
}

func (c *GlobalCache) SetGlob(globExpr string, glob *regexp.Regexp) {
	c.GlobLock.Lock()
	c.Glob[globExpr] = glob
	c.GlobLock.Unlock()
}

func (c *GlobalCache) GetGlob(glob string) *regexp.Regexp {
	c.GlobLock.RLock()
	defer c.GlobLock.RUnlock()
	return c.Glob[glob]
}

type Operation struct {
	Record
	Op        string `json:"op"`
	Domain    string `json:"domain"`
	Subdomain string `json:"subdomain"`

	Registry string
	Role     string
}

func MatchGlob(str, glob string, cache *GlobalCache) (_ bool, err error) {
	switch {
	case glob == "":
		return str == "", nil
	case glob == "*":
		return true, nil
	case glob == ".*":
		return true, nil
	}

	compiled := cache.GetGlob(glob)
	if compiled == nil {
		compiled, err = regexp.Compile(glob)
		if err != nil {
			return false, err
		}
		cache.SetGlob(glob, compiled)
	}

	return compiled.MatchString(str), nil
}

func ValidateAll(ctx context.Context, cache *GlobalCache, db *Database, role string, operations []*Operation) (error, error) {
	for _, op := range operations {
		subdomainGlob, ok, err := db.QueryRoleSubdomainGlob(ctx, role, op.Domain)
		if err != nil {
			return err, nil
		}
		if !ok {
			return fmt.Errorf("permission denied: %s is not under control of the role %s", op.Domain, role), nil
		}

		op.Domain, err = idna.ToASCII(op.Domain)
		if err != nil {
			return err, nil
		}

		if op.Subdomain == "" {
			op.CanonicalName = op.Domain
		} else {
			op.Subdomain, err = idna.ToASCII(op.Subdomain)
			if err != nil {
				return err, nil
			}

			ok, err := MatchGlob(op.Subdomain, subdomainGlob, cache)
			if err != nil {
				return err, nil
			}
			if !ok {
				return fmt.Errorf("operation not permitted: %s breaks the glob pattern for role %s: `%s`", op.Subdomain, role, subdomainGlob), nil
			}

			op.CanonicalName = op.Subdomain + "." + op.Domain
		}

		op.Role = role
	}

	return nil, nil
}

func BuildRegistries(ctx context.Context, db *Database, operations []*Operation) (map[string]Registry, error) {
	registries := make(map[string]Registry)

	for _, op := range operations {
		if registries[op.Registry] != nil {
			continue
		}

		registryName, ok, err := db.QueryDomainRegistry(ctx, op.Domain)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("no corresponding registry for domain %s", op.Domain)
		}

		conf, err := db.QueryRegistryConfig(ctx, registryName)
		switch {
		case err != nil:
			return nil, err
		case conf == nil:
			return nil, fmt.Errorf("missing config for registry %s", registryName)
		}

		builderName, ok := conf["builder"]
		if !ok {
			return nil, fmt.Errorf("undefined registry builder for %s: key `builder` should be specified", op.Domain)
		}

		registryBuilder, ok := RegistryBuilders[builderName]
		if !ok {
			return nil, fmt.Errorf("no registry builder called %s found for %s", builderName, op.Domain)
		}

		registry, err := registryBuilder(conf)
		if err != nil {
			return nil, err
		}

		registries[registryName] = registry
		op.Registry = registryName
	}

	return registries, nil
}

func ExecuteAll(operations []*Operation, registries map[string]Registry, callback func(err error, op *Operation)) {
	deleted := map[string][]*Operation{}
	updated := map[string][]*Operation{}

	for _, op := range operations {
		switch op.Op {
		case OP_DELETE:
			deleted[op.Registry] = append(deleted[op.Registry], op)
		case OP_UPDATE:
			updated[op.Registry] = append(updated[op.Registry], op)
		}
	}

	haveDone := map[string]bool{}

	var wg sync.WaitGroup
	for registryName, operations := range updated {
		for _, op := range operations {
			if haveDone[op.Domain] {
				continue
			}
			haveDone[op.Domain] = true

			wg.Add(1)
			go func() {
				defer wg.Done()
				err := registries[registryName].DeleteAllRecordsWithDomain(op.CanonicalName)
				if err != nil {
					callback(fmt.Errorf("deleting all records with domain [%s] failed: %v", op.Domain, err), op)
				}
			}()
		}
	}
	wg.Wait()

	for registryName, operations := range updated {
		for _, op := range operations {
			go func() {
				err := registries[registryName].AppendRecord(&op.Record)
				callback(err, op)
			}()
		}
	}

	for registryName, operations := range deleted {
		for _, op := range operations {
			go func() {
				err := registries[registryName].DeleteRecord(&op.Record)
				callback(err, op)
			}()
		}
	}
}
