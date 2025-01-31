# autodns

Centralized DNS management agent.

# Server

## Configuration

All configurations are stored in Redis database for the co-management with external programs without restarting.

Configuration can be done with inspection tools such as `redis-cli` and Redis Insight.

| Redis Key                  | Type     | Description                                                                                                          |
|----------------------------|----------|----------------------------------------------------------------------------------------------------------------------|
| `Registry:<domain>`        | Hash Map | Store key values that will be taken by the corresponding registry operator builder which specified by key `builder`. |
| `RoleToken:<role>`         | Hash Map | Token as the key and description as the value.                                                                       |
| `RoleSubdomainGlob:<role>` | Hash Map | Glob patterns of domains managed by the role. Only the updates with matched subdomain will be accepted.              |
| `DomainRegistry`           | Hash Map | Specify the corresponding registry of the domain.                                                                    |

### Example: Configure via Redis

```redis
hset Registry:jellyterra.com builder cloudflare
hset Registry:jellyterra.com api_token --
hset Registry:jellyterra.com zone jellyterra.com

hset DomainRegistry host-1.jellyterra.com jellyterra.com

hset RoleSubdomainGlob:jellyterra host-1.jellyterra.com *

hset RoleToken:jellyterra -- host-1
```

## Registry

Builtin registry builders are defined in `cmd/autodnsctl/import.go`

Supported in mainline:

| Name       | Registry   |
|------------|------------|
| cloudflare | Cloudflare |

### cloudflare

| Key         | Value             |
|-------------|-------------------|
| `builder`   | `cloudflare`      |
| `api_token` | API Token.        |
| `zone`      | Name of the zone. | 

## System Service

### systemd

```ini
[Unit]
Description=AutoDNS Server
After=redis.service

[Service]
Type=simple
ExecStart=/bin/autodnsctl --server --redis-addr [::1]:6379 --redis-db 5 --http-listen :5380
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

# DDNS Client

It will reload the configuration if it has changed when it is triggered by notification.
It exits on failure of loading.

Updating won't occur if the addresses in address sets and the configuration file have not changed.

## Configuration

- `addr_sets` Contains a sets of address filter rule with name.
    - `name` Is the identifier of the set of address.
    - `interfaces` OS network interface name. Non-existing one will be ignored.
    - `rules`: Rule for address filtering.
        - `pass`: Pass to the next rule when matched, or not.
        - `glob`: The glob pattern for address filtering.
- `zones` Groups of domains managed by different AutoDNS servers.
    - `server` The AutoDNS server URI prefix.
    - `role` The role for operation.
    - `token` The token as credential of the role.
    - `records` Domain records to update.
        - `domain` Domain name.
        - `subdomain` Subdomain name.
        - `ttl` TTL time in second.
        - `addr_sets` Address sets.

### Example

```json
{
  "addr_sets": [
    {
      "name": "v6",
      "interfaces": [
        "enp1s0",
        "enp2s0"
      ],
      "rules": [
        {
          "pass": true,
          "glob": "^2.*:.*"
        }
      ]
    },
    {
      "name": "v4",
      "interfaces": [
        "enp1s0",
        "enp2s0"
      ],
      "rules": [
        {
          "pass": false,
          "glob": "192.*"
        },
        {
          "pass": false,
          "glob": "10.*"
        }
      ]
    }
  ],
  "zones": [
    {
      "server": "http://[::1]:5380/<prefix>",
      "role": "<role>",
      "token": "<token>",
      "records": [
        {
          "domain": "host.registry-a.com",
          "subdomain": "edge-a",
          "ttl": 3600,
          "addr_sets": [
            "v6",
            "v4"
          ]
        }
      ]
    },
    {
      "server": "https://api.example.com/autodns/",
      "role": "<role>",
      "token": "<token>",
      "records": [
        {
          "domain": "host.registry-b.com",
          "subdomain": "edge-a",
          "ttl": 3600,
          "addr_sets": [
            "v6",
            "v4"
          ]
        }
      ]
    }
  ]
}
```

## System Service

### systemd

```ini
[Unit]
Description=AutoDNS DDNS Client
After=network.target

[Service]
Type=simple
ExecStart=/bin/autodnsctl --ddns --ddns-config <location>
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

# License

Copyright 2025 Jelly Terra <jellyterra@symboltics.com>

This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0
that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.
