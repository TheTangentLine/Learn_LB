# Admin API

The Admin API is an HTTP server that runs on a separate port (default `:9090`) and allows runtime management of backend pools without restarting the load balancer.

All state changes are serialised through the `Orchestrator`, which holds the appropriate `sync.RWMutex` locks. The Admin API itself has no additional locking concerns.

---

## Base URL

```
http://<admin-addr>
```

The admin address is set with the `--admin` flag (default: `:9090`).

---

## Endpoints

### `POST /admin/backends`

Register a new backend server into a ring.

If the named ring does not yet exist, it is created automatically. Adding a backend to a ring that already contains the same address is an error.

#### Request

**Headers**

| Header | Value |
|---|---|
| `Content-Type` | `application/json` |

**Body**

```json
{
  "ring": "<ring-type>",
  "addr": "<host:port>"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `ring` | string | yes | The ring to add the backend to. Must match a known `RingType` (e.g. `"api"`, `"default"`). An unknown value creates a new ring with that name. |
| `addr` | string | yes | The backend address in `host:port` form. This is the address the reverse proxy will forward requests to. |

**Example**

```bash
curl -X POST http://localhost:9090/admin/backends \
  -H "Content-Type: application/json" \
  -d '{"ring":"api","addr":"10.0.0.1:8081"}'
```

#### Responses

| Status | Meaning |
|---|---|
| `201 Created` | Backend was successfully registered. |
| `400 Bad Request` | Missing or malformed JSON body, or `ring`/`addr` fields are empty. |
| `409 Conflict` | The address is already registered in the specified ring. |

---

### `DELETE /admin/backends`

Remove a backend server from a ring.

The backend is immediately removed from the consistent hashing ring. Any in-flight requests that have already been proxied to that backend are not affected — they complete normally. New requests will no longer be routed to it.

#### Request

**Headers**

| Header | Value |
|---|---|
| `Content-Type` | `application/json` |

**Body**

```json
{
  "ring": "<ring-type>",
  "addr": "<host:port>"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `ring` | string | yes | The ring the backend belongs to. |
| `addr` | string | yes | The backend address to remove. |

**Example**

```bash
curl -X DELETE http://localhost:9090/admin/backends \
  -H "Content-Type: application/json" \
  -d '{"ring":"api","addr":"10.0.0.1:8081"}'
```

#### Responses

| Status | Meaning |
|---|---|
| `204 No Content` | Backend was successfully removed. |
| `400 Bad Request` | Missing or malformed JSON body, or `ring`/`addr` fields are empty. |
| `404 Not Found` | The specified ring does not exist. |

---

## Common workflows

### Bootstrap with initial backends (startup flag)

For backends known at deploy time, use the `--backends` flag instead of hitting the Admin API manually:

```bash
./lb \
  --listen  :8080 \
  --admin   :9090 \
  --backends "api:10.0.0.1:8081,api:10.0.0.2:8081,default:10.0.0.3:8082"
```

The flag format is a comma-separated list of `ring:host:port` entries. Splitting happens on the **first** colon, so IPv6 addresses or host-only entries with a port still parse correctly as long as the ring name has no colons.

### Rolling backend replacement

To swap a backend without dropping traffic:

1. Add the new backend:
   ```bash
   curl -X POST http://localhost:9090/admin/backends \
     -d '{"ring":"api","addr":"10.0.0.4:8081"}'
   ```
2. Verify the new node is healthy.
3. Remove the old backend:
   ```bash
   curl -X DELETE http://localhost:9090/admin/backends \
     -d '{"ring":"api","addr":"10.0.0.1:8081"}'
   ```

Because consistent hashing is used, step 1 only remaps the clients whose hash falls between the old last node and the new node — the rest of the fleet is unaffected.

### Draining a ring entirely

Remove backends one by one. Once the last backend is removed, the ring still exists in memory but is empty. Any request routed to that ring will receive a `503 Service Unavailable` response (or fall back to the `default` ring if one is populated).

---

## Error response format

All error responses are plain text with a trailing newline, as produced by Go's `http.Error`:

```
<error message>\n
```

For example, a 409 for a duplicate add returns:

```
server "10.0.0.1:8081" already in ring
```

---

## Security considerations

The Admin API has **no authentication**. It is intended to run on an internal/management network only. In production deployments:

- Bind it to a loopback or private interface (`--admin 127.0.0.1:9090`).
- Place it behind a reverse proxy that enforces mTLS or API key authentication.
- Use firewall rules to restrict access to the admin port from untrusted sources.
