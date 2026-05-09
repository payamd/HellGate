# Docker — single HellGate container (TCP + UDP on **9443**)

One process handles **SOCKS TCP** and **`udp://`** sessions on the same `POST /tunnel`. Use [`docker-compose.yml`](../docker-compose.yml) in the repo root:

```bash
cp server_config.example.json server_config.json
# edit tunnel_key; leave upstream_proxy empty for UDP to work
docker compose up -d --build
```

Firewall: **`9443/tcp`** (Apps Script reaches your relay over HTTP/TCP). In-app **UDP** targets are demuxed **inside** that encrypted stream at the exit — you do not need a host UDP port for the tunnel.

Apps Script [`Code.gs`](../apps_script/Code.gs): set `RELAY_URL` to `http://YOUR_HOST:9443/tunnel`.

The new **Demonica UDP** Android flavor is meant to pair with this relay (same TLS-to-Google path; exit on **9443**).
