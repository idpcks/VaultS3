# VaultS3 Helm Chart

Deploy [VaultS3](https://github.com/Kodiqa-Solutions/VaultS3) — a lightweight,
S3-compatible object store with a built-in dashboard — to Kubernetes.

VaultS3 runs as a **StatefulSet** with persistent volumes for object data
(`/data`) and BoltDB metadata (`/metadata`). A single port (`9000`) serves the
S3 API, the web dashboard (`/dashboard/`), Prometheus metrics (`/metrics`), and
the `/health` / `/ready` probe endpoints.

## Install

```bash
# From a local checkout of the repo:
helm install vaults3 ./deploy/helm/vaults3 \
  --namespace vaults3 --create-namespace \
  --set auth.secretKey="$(openssl rand -hex 20)"
```

Get the admin credentials and reach the dashboard:

```bash
kubectl -n vaults3 get secret vaults3 -o jsonpath='{.data.access-key}' | base64 -d; echo
kubectl -n vaults3 get secret vaults3 -o jsonpath='{.data.secret-key}' | base64 -d; echo
kubectl -n vaults3 port-forward svc/vaults3 9000:9000
# open http://localhost:9000/dashboard/
```

## Key values

| Key | Default | Description |
|-----|---------|-------------|
| `replicaCount` | `1` | Keep at 1 unless using Raft clustering (Beta). |
| `image.repository` | `eniz1806/vaults3` | Image repo. |
| `image.tag` | `""` | Defaults to the chart `appVersion`. Pin for reproducibility. |
| `auth.accessKey` | `vaults3-admin` | Admin access key (injected via Secret → env). |
| `auth.secretKey` | `vaults3-secret-change-me` | Admin secret. **Change it**, or set empty to auto-generate. |
| `auth.existingSecret` | `""` | Use your own Secret (keys `access-key`, `secret-key`). |
| `config` | single-node config | The `vaults3.yaml` mounted at `/etc/vaults3/`. Replace to enable encryption/replication/erasure/etc. |
| `existingConfigMap` | `""` | Use your own ConfigMap (key `vaults3.yaml`). |
| `persistence.enabled` | `true` | Keep enabled for real use. |
| `persistence.data.size` | `50Gi` | Object-data PVC size. |
| `persistence.metadata.size` | `5Gi` | Metadata PVC size. |
| `service.type` | `ClusterIP` | Use `LoadBalancer` or an Ingress to expose. |
| `ingress.enabled` | `false` | Enable + set `hosts` to expose via Ingress. For large uploads set `nginx.ingress.kubernetes.io/proxy-body-size: "0"`. |
| `serviceMonitor.enabled` | `false` | Prometheus-Operator scraping of `/metrics`. |
| `resources` | 100m/128Mi → 1/512Mi | VaultS3 is light (<80MB RAM typical). |
| `extraEnv` | `[]` | Extra env vars (`VAULTS3_LOG_LEVEL`, `VAULTS3_DOMAIN`, `VAULTS3_ENCRYPTION_KEY`, …). |

See [`values.yaml`](./values.yaml) for the full list.

## Enabling features

VaultS3 features (encryption, compression, replication, erasure coding, tiering,
etc.) are driven by `vaults3.yaml`. Override the `config` value with your own:

```bash
helm install vaults3 ./deploy/helm/vaults3 -n vaults3 --create-namespace \
  --set-file config=./my-vaults3.yaml \
  --set auth.secretKey="$(openssl rand -hex 20)"
```

Keep `storage.data_dir: /data` and `storage.metadata_dir: /metadata` (the chart
forces these via env vars anyway, so they always land on the PVCs).

## Clustering (Beta)

Multi-node Raft clustering is experimental. The default chart deploys an
independent single-node store per replica; running `replicaCount > 1` does **not**
form a cluster on its own. Cluster bootstrap/join needs additional `cluster:` and
inter-node port configuration — see
[`docs/SCALING.md`](https://github.com/Kodiqa-Solutions/VaultS3/blob/main/docs/SCALING.md).
For production today, prefer a single node with erasure coding for disk
redundancy.

## Uninstall

```bash
helm uninstall vaults3 -n vaults3
# PVCs are retained by design — delete them explicitly to wipe data:
kubectl -n vaults3 delete pvc -l app.kubernetes.io/name=vaults3
```
