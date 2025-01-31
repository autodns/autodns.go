// Copyright 2025 Jelly Terra <jellyterra@symboltics.com>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"fmt"
	"github.com/autodns/autodns.go/core"
	"github.com/redis/rueidis"
	"path"
)

func _server() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbClient, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{*redisAddr},
		SelectDB:    *redisIdx,
	})
	if err != nil {
		return err
	}

	fmt.Println("Test connection to Redis")

	err = dbClient.Do(ctx, dbClient.B().Ping().Build()).Error()
	if err != nil {
		return err
	}

	db := &core.Database{
		Client: dbClient,
	}

	go func() {
		<-signalC
		cancel()
	}()

	fmt.Println("Listen and serve on http://" + path.Join(*httpListen, *httpPrefix) + "/")

	return Serve(ctx, globalCache, *httpListen, *httpPrefix, db)
}
