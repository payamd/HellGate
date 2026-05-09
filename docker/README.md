# Docker — **9443** only (`hellgate`)

This compose file runs **one** container: **HellGate on TCP 9443** for the new stack (TCP + in-tunnel UDP). If you also run a legacy exit on **8443**, manage that container separately.

Config path on the host: **`./config/server_config.json`** (next to `docker-compose.yml`). Create the folder if needed.

```bash
mkdir -p config
cp server_config.example.json config/server_config.json
# edit tunnel_key; leave upstream_proxy "" for UDP egress
docker compose up -d --build
```

Firewall: **`9443/tcp`**. Apps Script hits the relay over **HTTP on TCP**; UDP workloads are carried inside the encrypted batch stream.

**`RELAY_URL`:** `http://YOUR_HOST:9443/tunnel` — must match **`server_port`** in `server_config.json`.
