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

### Private GitHub repo (clone inside container — Pat)

If the server uses the **golang + `git clone`** pattern and **`HellGate` is private**, anonymous clone fails (`could not read Username for 'https://github.com'`). Use a PAT without committing it:

1. GitHub → **Fine-grained token** or classic PAT → **Contents: Read** on `HellGate`.
2. On the VPS:

   ```bash
   cd /root
   printf '%s\n' 'GITHUB_TOKEN=ghp_REPLACE_ME' >> .env
   chmod 600 .env
   ```

3. Paste the **`hellgate`** service from [`docker-compose.git-private.example.yml`](../docker-compose.git-private.example.yml) into **`/root/docker-compose.yml`**, fixing the **`volumes`** path if **`HellGate/config`** isn’t `./HellGate/config` relative to that file.

   ```bash
   docker compose --env-file /root/.env up -d --build hellgate
   ```

Treat **`GITHUB_TOKEN`** like root access; rotate if exposed. Prefer **`build: { context: ./HellGate }`** when the repo already exists on disk so no token enters the clone URL path.
