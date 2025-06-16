// Copyright 2025 Jelly Terra <jellyterra@proton.me>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package core

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type RegistryDef struct {
	Builder       string            `json:"builder"`
	BuilderParams map[string]string `json:"builder_params"`
}

type ManagedDomainDef struct {
	Registry string `json:"registry"`

	Glob string `json:"glob"`
}

type AuthKeyDef struct {
	Expire int64 `json:"expiration_time"`
}

type RoleDef struct {
	Keys           map[string]AuthKeyDef       `json:"keys"`
	ManagedDomains map[string]ManagedDomainDef `json:"managed_domains"`
}

type ContextCache struct {
	Time     int64
	Val      any
	lastUsed atomic.Int64
}

type Context struct {
	BaseDir string

	CacheLifetime int64
	lastCheck     atomic.Int64

	Cache     map[string]*ContextCache
	cacheLock sync.RWMutex
}

func (c *Context) purgeCache() {
	now := time.Now().Unix()
	if now < c.lastCheck.Load()+c.CacheLifetime {
		return
	}

	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	// Check twice. The other thread might have gotten the lock and finished the job.
	if now < c.lastCheck.Load()+c.CacheLifetime {
		return
	}
	c.lastCheck.Store(now)

	for k, v := range c.Cache {
		if now > v.lastUsed.Load()+c.CacheLifetime {
			// Expired.
			delete(c.Cache, k)
			continue
		}

		_, err := os.Stat(k)
		if err != nil {
			// Deleted.
			delete(c.Cache, k)
		}
	}
}

func Query[T any](c *Context, v *T, keys ...string) (*T, error) {
	c.purgeCache()

	fName := path.Join(keys...) + ".json"
	if strings.Contains(fName, "..") {
		return nil, errors.New("invalid fName")
	}

	fStat, err := os.Stat(fName)
	if err != nil {
		return nil, err
	}

	cacheMiss := func() (*T, error) {
		b, err := os.ReadFile(fName)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(b, v)
		if err != nil {
			return nil, err
		}

		// Write to cache.
		c.cacheLock.Lock()
		cache := &ContextCache{Time: fStat.ModTime().Unix(), Val: v}
		cache.lastUsed.Store(time.Now().Unix())
		c.Cache[fName] = cache
		c.cacheLock.Unlock()

		return v, nil
	}

	c.cacheLock.RLock()
	cache, exist := c.Cache[fName]
	c.cacheLock.RUnlock()
	if exist {
		// Cache miss.
		if fStat.ModTime().Unix() != cache.Time {
			// Disk change.
			v, err = cacheMiss()
			if err != nil {
				return nil, err
			}
		} else {
			// Cache hit.
			cache.lastUsed.Store(time.Now().Unix())
			v = cache.Val.(*T)
		}
	} else {
		v, err = cacheMiss()
		if err != nil {
			return nil, err
		}
	}

	return v, err
}

type ValidationResult struct {
	Registry string
}

func Validate(roleDef *RoleDef, domain string, subdomain string) (*ValidationResult, error) {
	d, exist := roleDef.ManagedDomains[domain]
	if !exist {
		return nil, errors.New("permission denied")
	}

	switch d.Glob {
	case "":
		if subdomain != "" {
			return nil, errors.New("permission denied")
		}
	case "*":
	default:
		matched, err := regexp.MatchString(d.Glob, subdomain)
		if err != nil {
			return nil, err
		}
		if !matched {
			return nil, errors.New("permission denied")
		}
	}

	return &ValidationResult{
		Registry: d.Registry,
	}, nil
}
