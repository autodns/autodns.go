// Copyright 2025 Jelly Terra <jellyterra@symboltics.com>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"fmt"
	"github.com/autodns/autodns.go/core"
	"github.com/redis/rueidis"
)

func _operator() error {
	ctx := context.Background()

	dbClient, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{*redisAddr},
		SelectDB:    *redisIdx,
	})
	if err != nil {
		return fmt.Errorf("ping redis server: %v", err)
	}

	db := &core.Database{
		Client: dbClient,
	}

	err = fmt.Errorf("nothing to do")

	switch {
	case *role != "":
		switch {
		case *domain != "":
			switch {
			case *glob != "":
				err := db.UpdateRoleSubdomainGlob(ctx, *role, *domain, *glob)
				if err != nil {
					return err
				}
				fmt.Println("Role", *role, "has gotten control of", *domain, "when the subdomain matches pattern", *glob)
			case *opQuery:
				v, ok, err := db.QueryRoleSubdomainGlob(ctx, *role, *domain)
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("domain %s is not under control of the role %s", *domain, *role)
				}
				fmt.Println(v)
			case *opDelete:
				err := db.DeleteRoleSubdomainGlob(ctx, *role, *domain)
				if err != nil {
					return err
				}
				fmt.Println("Role", *role, "has been revoked control of", *domain)
			}
		case *token != "":
			switch {
			case *desc != "":
				err := db.UpdateRoleToken(ctx, *role, *token, *desc)
				if err != nil {
					return err
				}
				fmt.Println("Token assigned.")
			case *opDelete:
				err := db.DeleteRoleToken(ctx, *role, *token)
				if err != nil {
					return err
				}
				fmt.Println("Token revoked.")
			}
		case *opDelete:
			err := db.DeleteRole(ctx, *role)
			if err != nil {
				return err
			}
			fmt.Println("Role deleted.")
		}
	case *registry != "":
		switch {
		case *key != "" && *val != "":
			err := db.Do(ctx, db.B().Hset().Key(core.PREFIX_REGISTRY+*registry).FieldValue().FieldValue(*key, *val).Build()).Error()
			if err != nil {
				return err
			}
			fmt.Println("Set registry configuration of", *registry, "key", *key, "with value", *val)
		case *opDelete:
			err := db.DeleteRegistry(ctx, *registry)
			if err != nil {
				return err
			}
			fmt.Println("Registry configuration deleted.")
		}
	}

	if err != nil {
		fmt.Println("Nothing to do. For more operations please operate Redis database with inspection tools such as redis-cli and Redis Insight.")
	}

	return nil
}
