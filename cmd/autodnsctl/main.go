// Copyright 2025 Jelly Terra <jellyterra@symboltics.com>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

import (
	"flag"
	"fmt"
	"github.com/autodns/autodns.go/core"
	"os"
	"os/signal"
	"syscall"
)

var (
	asDDNS = flag.Bool("ddns", false, "Run as DDNS client.")

	ddnsConfig = flag.String("ddns-config", "", "DDNS configuration file location.")

	asServer = flag.Bool("server", false, "Run as server.")

	httpListen = flag.String("http-listen", "", "HTTP listen address.")
	httpPrefix = flag.String("http-prefix", "/", "HTTP REST API prefix.")
	redisAddr  = flag.String("redis-addr", "[::1]:6379", "Redis address.")
	redisIdx   = flag.Int("redis-db", 0, "Redis database index.")

	asOperator = flag.Bool("operate", false, "Run as operator.")

	opQuery  = flag.Bool("query", false, "Query.")
	opDelete = flag.Bool("delete", false, "Delete.")

	role     = flag.String("role", "", "Specify the role.")
	registry = flag.String("registry", "", "Specify the registry.")
	domain   = flag.String("domain", "", "Specify the domain.")

	token = flag.String("token", "", "Specify the token.")
	desc  = flag.String("desc", "", "Set description.")
	glob  = flag.String("glob", "", "Set glob pattern.")
	key   = flag.String("key", "", "Specify the key.")
	val   = flag.String("value", "", "Set the value.")

	triggerTime = flag.Int("trigger-time", 0, "Set the update duration in second when NETLINK is unavailable.")

	globalCache = core.NewGlobalCache()

	signalC = make(chan os.Signal, 1)
)

func main() {
	flag.Parse()

	var err error

	signal.Notify(signalC, syscall.SIGINT, syscall.SIGTERM)

	switch {
	case *asOperator:
		err = _operator()
	case *asDDNS:
		err = _ddns()
	case *asServer:
		err = _server()
	}
	if err != nil {
		fmt.Println(err)
	}
}
