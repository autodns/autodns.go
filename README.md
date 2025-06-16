# autodns

Centralized DNS management agent.

## Usage

```
$ autodnsctl --help
Usage of autodnsctl :
Subcommands:
        serve           Run as server.
        ddns            Run as DDNS client.
        server-config   Configure.
Learn more via subcommand with option --help
```

```
$ autodnsctl serve --help
Usage of serve:
  -cache-lifetime int
        Cache lifetime in seconds. (default 3600)
  -config-dir string
        Base directory for reading config in JSON. (default ".")
  -http-addr string
        HTTP listen address. (default ":5380")
  -http-route string
        HTTP route. (default "/")
```

```
$ autodnsctl ddns --help
Usage of ddns:
  -config string
        Path to DDNS config file. (default "./ddns.json")
  -trigger-duration int
        Duration to trigger DNS update. Duration in seconds. (default 30)
```

```
$ autodnsctl server-config --help
Usage of server-config:
  -builder string
        Builder name.
  -builder-param-key string
        Builder params key
  -builder-param-val string
        Builder params key
  -config-dir string
        Base directory for storing config in JSON. (default ".")
  -create-domain-delegation
        Delegate domain.
  -create-key
        Create key.
  -create-registry
        Create registry.
  -create-role
        Create role.
  -delete-key
        Delete key.
  -delete-registry
        Delete registry.
  -delete-role
        Delete role.
  -domain string
        Domain name.
  -expire-at int
        Expiration time in Unix epoch. Zero value to be never.
  -glob string
        Glob pattern. Empty to be **the same only**.
  -key string
        Key name.
  -registry string
        Registry name.
  -revoke-domain-delegation
        Revoke domain delegation.
  -role string
        Role name.
  -set-builder-param
        Set builder param.
```

# Server

## Configuration

All configurations are stored in JSON on filesystem for the co-management with external programs without restarting.

### Cache

Server will cache each configuration file that has been read.
When reloading finds that the file has changed, the cache will be purged.
Caches that exceed their lifetime will be purged.

### Example

```shell
cd /var/lib/autodns/

# Create registry.
autodnsctl server-config --registry jellyterra.com --create-registry --builder cloudflare
autodnsctl server-config --registry jellyterra.com --set-builder-param --builder-param-key 'zone' --builder-param-val 'jellyterra.com'
autodnsctl server-config --registry jellyterra.com --set-builder-param --builder-param-key 'api_token' --builder-param-val '<API Token>'

# Create role.
autodnsctl server-config --role jellyterra --create-role
autodnsctl server-config --role jellyterra --create-key --key 'abc123' --expire-at 1750061600

# Delegate domain control to the role.
autodnsctl server-config --role jellyterra --registry jellyterra.com --create-domain-delegation --domain hosts.jellyterra.com --glob '*' # of *.hosts.jellyterra.com
autodnsctl server-config --role jellyterra --registry jellyterra.com --create-domain-delegation --domain cdn.jellyterra.com --glob '' # of cdn.jellyterra.com

# Serve!
autodnsctl serve
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

[Service]
Type=simple
ExecStart=autodnsctl serve --config-dir /var/lib/autodns/ --http-addr :5380 --http-route /
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
    - `key` The key as credential of the role.
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
      "server": "https://<Server Addr>/<HTTP Route>/",
      "role": "<Role>",
      "key": "<Key>",
      "records": [
        {
          "domain": "hosts.jellyterra.com",
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
ExecStart=autodnsctl ddns --config /etc/ddns.json
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

# License

**Copyright 2025 Jelly Terra <jellyterra@proton.me>**

This Source Code Form is subject to the terms of the **Mozilla Public License, v. 2.0**
that can be found in the LICENSE file and https://mozilla.org/MPL/2.0/.
