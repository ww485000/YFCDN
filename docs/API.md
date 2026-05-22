# YFCDN API

Base path: `/api/v1`.

## Auth

### Login

`POST /api/v1/auth/login`

```json
{"username":"admin","password":"admin123"}
```

Response contains a `token`. Use it as:

```text
Authorization: Bearer SESSION_TOKEN
```

API keys can also authenticate requests:

```text
X-YFCDN-API-Key: yfc_xxx
```

## Core resources

All authenticated resources support JSON request and response bodies.

| Resource | List/Create | Item |
| --- | --- | --- |
| Tenants | `GET/POST /tenants` | `GET/PUT/DELETE /tenants/{id}` |
| Operators | `GET/POST /operators` | `GET/PUT/DELETE /operators/{id}` |
| Nodes | `GET/POST /nodes` | `GET/PUT/DELETE /nodes/{id}` |
| Domains | `GET/POST /domains` | `GET/PUT/DELETE /domains/{id}` |
| Plans | `GET/POST /plans` | `GET/PUT/DELETE /plans/{id}` |
| API keys | `GET/POST /api-keys` | `DELETE /api-keys/{id}` |

## Agent endpoints

The edge agent authenticates with `X-Node-Key`.

### Heartbeat

`POST /api/v1/agent/heartbeat`

```json
{
  "cpu_percent": 12.5,
  "load1": 0.3,
  "memory_used": 536870912,
  "memory_total": 2147483648,
  "rx_bytes": 123456,
  "tx_bytes": 456789
}
```

### Pull config

`GET /api/v1/agent/config`

Response contains the node object, active domains assigned to that node, and generated Nginx config.

## Node Nginx preview

`GET /api/v1/nodes/{id}/nginx`

Returns the generated Nginx config for a node. Useful before enabling reloads on production edge nodes.
