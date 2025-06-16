// Copyright 2025 Jelly Terra <jellyterra@proton.me>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"fmt"
	"github.com/autodns/autodns.go/core"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"
)

type Rule struct {
	Pass bool   `json:"pass"`
	Glob string `json:"glob"`

	CompiledGlob *regexp.Regexp
}

type AddrSet struct {
	Name       string   `json:"name"`
	Interfaces []string `json:"interfaces"`
	Rules      []*Rule  `json:"rules"`
}

type Record struct {
	Domain    string   `json:"domain"`
	Subdomain string   `json:"subdomain"`
	TTL       int      `json:"ttl"`
	AddrSets  []string `json:"addr_sets"`
}

type Zone struct {
	Server  string   `json:"server"`
	Role    string   `json:"role"`
	Key     string   `json:"key"`
	Records []Record `json:"records"`
}

type DDNSConfig struct {
	AddrSets []AddrSet `json:"addr_sets"`
	Zones    []Zone    `json:"zones"`
}

func DDNS(configPath string) func() error {
	var (
		lastModTime   = time.Now().Unix()
		config        *DDNSConfig
		addrSetsCache map[string]map[string]bool
	)

	// Drop config cache and start from none.
	initAddrSetsCache := func() {
		addrSetsCache = map[string]map[string]bool{}
		for _, addrSet := range config.AddrSets {
			addrSetsCache[addrSet.Name] = map[string]bool{}
		}
	}

	// Update records.
	return func() error {
		stat, err := os.Stat(configPath)
		if err != nil {
			return err
		}

		if stat.ModTime().Unix() != lastModTime {
			fmt.Println("Load configuration")
			config, err = LoadDDNSConfig(configPath)
			if err != nil {
				return fmt.Errorf("loading config in JSON failed: %v", err)
			}

			lastModTime = stat.ModTime().Unix()

			initAddrSetsCache()
		}

		addrSetMap := map[string][]net.IP{}

		same := true
		for _, addrSet := range config.AddrSets {
			addrs, err := CollectAddrSet(addrSet)
			if err != nil {
				return err
			}
			addrSetMap[addrSet.Name] = addrs

			for _, addr := range addrs {
				same = addrSetsCache[addrSet.Name][addr.String()] && same
			}
		}
		if same {
			return nil
		}

		initAddrSetsCache()
		for addrSetName, addrSet := range addrSetMap {
			for _, addr := range addrSet {
				addrSetsCache[addrSetName][addr.String()] = true
			}
		}

		for _, zone := range config.Zones {
			var operations []*core.Operation

			for _, record := range zone.Records {
				addrMap := map[string]net.IP{}

				for _, addrSetName := range record.AddrSets {
					for _, addr := range addrSetMap[addrSetName] {
						addrMap[addr.String()] = addr
					}
				}

				for _, addr := range slices.Collect(maps.Values(addrMap)) {
					var typ string

					switch {
					case addr.To4() != nil:
						typ = "A"
					case addr.To16() != nil:
						typ = "AAAA"
					default:
						continue
					}

					op := core.Operation{
						Record: core.Record{
							Type:  typ,
							Value: addr.String(),
							TTL:   record.TTL,
						},
						Op:        "update",
						Domain:    record.Domain,
						Subdomain: record.Subdomain,
					}
					if op.Subdomain == "" {
						op.CanonicalName = op.Domain
					} else {
						op.CanonicalName = op.Subdomain + "." + op.Domain
					}
					operations = append(operations, &op)
				}
			}

			go func() {
				u, err := url.JoinPath(zone.Server, "/v1/do")
				if err != nil {
					fmt.Println(err)
					return
				}

				resp, err := http.Post(u, "application/json", bytes.NewReader(MarshalJSON(&ReqDo{
					Role:       zone.Role,
					Token:      zone.Key,
					Operations: operations,
				})))
				if err != nil {
					fmt.Println(err)
				}
				if resp.StatusCode != http.StatusOK {
					b, err := io.ReadAll(resp.Body)
					if err != nil {
						fmt.Println(err)
						return
					}
					fmt.Println(string(b))
				}
			}()

			for _, op := range operations {
				fmt.Println("Update", op.CanonicalName, "=>", op.Value)
			}
		}

		return nil
	}
}

func LoadDDNSConfig(path string) (*DDNSConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config, err := UnmarshalJSON(data, &DDNSConfig{})
	if err != nil {
		return nil, err
	}

	for _, addrSet := range config.AddrSets {
		for _, rule := range addrSet.Rules {
			rule.CompiledGlob, err = regexp.Compile(rule.Glob)
			if err != nil {
				return nil, err
			}
		}
	}

	return config, nil
}

func CollectAddrSet(addrSet AddrSet) ([]net.IP, error) {
	var interfaces []*net.Interface

	for _, ifaceName := range addrSet.Interfaces {
		iface, err := net.InterfaceByName(ifaceName)
		if err != nil {
			continue
		}
		interfaces = append(interfaces, iface)
	}

	addrs, err := CollectInterfaces(interfaces)
	if err != nil {
		return nil, err
	}

	return FilterAddrsByRules(addrs, addrSet.Rules)
}

func CollectInterfaces(interfaces []*net.Interface) (collected []net.IP, _ error) {
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			split := strings.Split(addr.String(), "/")
			collected = append(collected, net.ParseIP(split[0]))
		}
	}
	return collected, nil
}

func FilterAddrsByRules(addrs []net.IP, rules []*Rule) (filtered []net.IP, err error) {
	for _, addr := range addrs {
		for _, rule := range rules {
			matched := rule.CompiledGlob.MatchString(addr.String())
			if rule.Pass && matched || !rule.Pass && !matched {
				filtered = append(filtered, addr)
			}
		}
	}
	return filtered, nil
}
