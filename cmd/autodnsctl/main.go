// Copyright 2025 Jelly Terra <jellyterra@proton.me>
// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
// that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/autodns/autodns.go/core"
	"log"
	"os"
	"os/signal"
	"path"
	"regexp"
	"syscall"
	"time"
)

var (
	signalC = make(chan os.Signal, 1)
)

func main() {
	signal.Notify(signalC, syscall.SIGINT, syscall.SIGTERM)

	err := _main()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func _main() error {
	flag.Usage = func() {
		fmt.Println("Usage of", os.Args[0], ":")
		fmt.Println("Subcommands:")
		fmt.Println("\tserve           Run as server.")
		fmt.Println("\tddns            Run as DDNS client.")
		fmt.Println("\tserver-config   Configure.")
		fmt.Println("Learn more via subcommand with option --help")
	}
	flag.Parse()

	if len(os.Args) < 2 {
		flag.Usage()
		return nil
	}

	switch os.Args[1] {
	case "serve":
		return _serve(os.Args[2:])
	case "ddns":
		return _ddns(os.Args[2:])
	case "server-config":
		return _server_config(os.Args[2:])
	default:
		return fmt.Errorf("unknown subcommand %s", os.Args[1])
	}
}

func _serve(args []string) error {
	f := flag.NewFlagSet("serve", flag.ExitOnError)
	var (
		baseDir   = f.String("config-dir", ".", "Base directory for reading config in JSON.")
		httpAddr  = f.String("http-addr", ":5380", "HTTP listen address.")
		httpRoute = f.String("http-route", "/", "HTTP route.")

		cacheLifetime = f.Int64("cache-lifetime", 3600, "Cache lifetime in seconds.")
	)
	_ = f.Parse(args)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-signalC
		cancel()
	}()

	fmt.Println("Listen and serve on http://" + path.Join(*httpAddr, *httpRoute) + "/")

	return Serve(ctx, &core.Context{
		BaseDir:       *baseDir,
		CacheLifetime: *cacheLifetime,
		Cache:         map[string]*core.ContextCache{},
	}, *httpAddr, *httpRoute)
}

func _ddns(args []string) error {
	f := flag.NewFlagSet("ddns", flag.ExitOnError)
	var (
		configPath      = f.String("config", "./ddns.json", "Path to DDNS config file.")
		triggerDuration = f.Int("trigger-duration", 30, "Duration to trigger DNS update. Duration in seconds.")
	)
	_ = f.Parse(args)

	triggerC := make(chan struct{}, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := Trigger(ctx, *triggerDuration, triggerC)
		if err != nil {
			log.Fatalln("notification trigger setup failed:", err)
		}
	}()

	triggerDDNS := DDNS(*configPath)

	triggerC <- struct{}{}

	for {
		select {
		case <-triggerC:
			delay := time.After(1 * time.Second)
			func() {
				for {
					select {
					case <-delay:
						return
					case <-triggerC:
					}
				}
			}()

			err := triggerDDNS()
			if err != nil {
				return err
			}
		case <-signalC:
			return nil
		}
	}
}

func _server_config(args []string) error {
	f := flag.NewFlagSet("server-config", flag.ExitOnError)
	var (
		baseDir = f.String("config-dir", ".", "Base directory for storing config in JSON.")

		role     = f.String("role", "", "Role name.")
		key      = f.String("key", "", "Key name.")
		expireAt = f.Int64("expire-at", 0, "Expiration time in Unix epoch. Zero value to be never.")
		domain   = f.String("domain", "", "Domain name.")
		glob     = f.String("glob", "", "Glob pattern. Empty to be **the same only**.")
		registry = f.String("registry", "", "Registry name.")

		createRole = f.Bool("create-role", false, "Create role.")
		deleteRole = f.Bool("delete-role", false, "Delete role.")
		createKey  = f.Bool("create-key", false, "Create key.")
		deleteKey  = f.Bool("delete-key", false, "Delete key.")

		createDomainDelegation = f.Bool("create-domain-delegation", false, "Delegate domain.")
		revokeDomainDelegation = f.Bool("revoke-domain-delegation", false, "Revoke domain delegation.")

		builder         = f.String("builder", "", "Builder name.")
		builderParamKey = f.String("builder-param-key", "", "Builder params key")
		builderParamVal = f.String("builder-param-val", "", "Builder params key")

		setBuilderParam = f.Bool("set-builder-param", false, "Set builder param.")

		createRegistry = f.Bool("create-registry", false, "Create registry.")
		deleteRegistry = f.Bool("delete-registry", false, "Delete registry.")
	)
	_ = f.Parse(args)

	var (
		rolePath     = path.Join(*baseDir, "role", *role+".json")
		registryPath = path.Join(*baseDir, "registry", *registry+".json")
	)

	switch {
	case *createRole:
		if *role == "" {
			return fmt.Errorf("requires [role]")
		}

		err := MarshalJSONToPath(rolePath, &core.RoleDef{
			Keys:           map[string]core.AuthKeyDef{},
			ManagedDomains: map[string]core.ManagedDomainDef{},
		})
		if err != nil {
			return err
		}

		fmt.Printf("Role [%s] created.\n", *role)
	case *deleteRole:
		if *role == "" {
			return fmt.Errorf("requires [role]")
		}

		err := os.Remove(rolePath)
		if err != nil {
			return err
		}

		fmt.Printf("Role [%s] removed.\n", *role)
	case *createKey:
		if *role == "" || *key == "" {
			return fmt.Errorf("requires [role, key], optional [expire-at]")
		}

		roleDef, err := UnmarshalJSONFromPath(rolePath, &core.RoleDef{})
		if err != nil {
			return err
		}

		roleDef.Keys[*key] = core.AuthKeyDef{
			Expire: *expireAt,
		}

		err = MarshalJSONToPath(rolePath, &roleDef)
		if err != nil {
			return err
		}

		fmt.Printf("Role [%s] key [%s] created and expires on [%d].\n", *role, *key, *expireAt)
	case *deleteKey:
		if *role == "" || *key == "" {
			return fmt.Errorf("requires [role, key]")
		}

		roleDef, err := UnmarshalJSONFromPath(rolePath, &core.RoleDef{})
		if err != nil {
			return err
		}

		delete(roleDef.Keys, *key)

		err = MarshalJSONToPath(rolePath, &roleDef)
		if err != nil {
			return err
		}

		fmt.Printf("Role [%s] key [%s] removed.\n", *role, *key)
	case *createDomainDelegation:
		if *role == "" || *domain == "" || *registry == "" {
			return fmt.Errorf("requires [role, domain, registry], optional [glob]")
		}

		roleDef, err := UnmarshalJSONFromPath(rolePath, &core.RoleDef{})
		if err != nil {
			return err
		}

		switch *glob {
		case "":
		case "*":
		default:
			_, err := regexp.Compile(*glob)
			if err != nil {
				return fmt.Errorf("validating glob pattern [%s]: %v", *glob, err)
			}
		}

		roleDef.ManagedDomains[*domain] = core.ManagedDomainDef{
			Registry: *registry,
			Glob:     *glob,
		}

		err = MarshalJSONToPath(rolePath, &roleDef)
		if err != nil {
			return err
		}

		fmt.Printf("Role [%s] has been delegated control of domain [%s] matching glob pattern [%s] under registry [%s].\n", *role, *domain, *glob, *registry)
	case *revokeDomainDelegation:
		if *role == "" || *domain == "" {
			return fmt.Errorf("requires [role, domain]")
		}

		roleDef, err := UnmarshalJSONFromPath(rolePath, &core.RoleDef{})
		if err != nil {
			return err
		}

		delete(roleDef.ManagedDomains, *domain)

		err = MarshalJSONToPath(rolePath, &roleDef)
		if err != nil {
			return err
		}

		fmt.Printf("Role [%s] has had delegation of control revoked for domain [%s].\n", *role, *domain)
	case *createRegistry:
		if *registry == "" || *builder == "" {
			return fmt.Errorf("requires [registry, builder]")
		}

		if core.RegistryBuilders[*builder] == nil {
			fmt.Printf("Warning: registry builder [%s] is not builtin and not available!\n", *builder)
		}

		err := MarshalJSONToPath(path.Join(*baseDir, "registry", *registry+".json"), &core.RegistryDef{
			Builder:       *builder,
			BuilderParams: map[string]string{},
		})
		if err != nil {
			return err
		}

		fmt.Printf("Registry [%s] using builder [%s] created.\n", *registry, *builder)
	case *deleteRegistry:
		err := os.Remove(registryPath)
		if err != nil {
			return err
		}

		fmt.Printf("Registry [%s] removed.\n", *registry)
	case *setBuilderParam:
		if *registry == "" || *builderParamKey == "" || *builderParamVal == "" {
			return fmt.Errorf("requires [registry, builder-param-key, builder-param-val]")
		}

		registryDef, err := UnmarshalJSONFromPath(registryPath, &core.RegistryDef{})
		if err != nil {
			return err
		}

		registryDef.BuilderParams[*builderParamKey] = *builderParamVal

		err = MarshalJSONToPath(registryPath, &registryDef)
		if err != nil {
			return err
		}

		fmt.Printf("Registry [%s] builder param [%s] has been set to [%s].\n", *registry, *builderParamKey, *builderParamVal)
	default:
		fmt.Println("Nothing to do. Check --help for more information.")
	}

	return nil
}
