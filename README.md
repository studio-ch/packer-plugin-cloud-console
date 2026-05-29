# packer-plugin

A [HashiCorp Packer](https://www.packer.io) builder plugin that builds VM
images by driving the **studio-cp tenant REST API** with a Bearer API key.

It registers a base image, boots a builder VM, provisions it over SSH, shuts
it down, and pushes the result as an OCI image — all through the public `/v1`
API surface. It has **no** dependency on the upstream xcloud Go module, gRPC,
mTLS, proto, or VNC.

> **Standalone module.** This folder is a self-contained Go module
> (`github.com/studio-ch/packer-plugin`) and can be installed directly with
> `go install github.com/studio-ch/packer-plugin@latest`. It has its own
> `go.mod` and is independent of any Node/pnpm workspace.

## Build

```bash
cd packer-plugin
go build ./...
```

To install for local Packer use:

```bash
go build -o ~/.packer.d/plugins/packer-plugin-studio-cp .
```

## Authentication

The plugin authenticates with a tenant API key sent as
`Authorization: Bearer <token>`. Mutations require the `write:resources`
scope (enforced server-side) and the `xcloud` service entitlement.

| Config field   | Environment fallback        |
| -------------- | --------------------------- |
| `api_endpoint` | `STUDIO_CP_API_ENDPOINT`    |
| `api_token`    | `STUDIO_CP_API_TOKEN`       |

## Configuration

| Field                | Type      | Default          | Notes |
| -------------------- | --------- | ---------------- | ----- |
| `api_endpoint`       | string    | — (required)     | API host, with or without scheme. |
| `api_token`          | string    | — (required)     | Tenant API key. |
| `region_id`          | string    | — (required)     | Region UUID. |
| `name`               | string    | `packer-<8hex>`  | Instance name. |
| `cpu_cores`          | int       | `4`              | |
| `memory`             | int       | `8`              | GiB. |
| `disk`               | int       | `64`             | GiB. |
| `network`            | string    | `default`        | Network name to attach. |
| `image`              | string    | —                | Existing catalog image (mutually exclusive with `pull_image`). |
| `pull_image`         | string    | —                | OCI reference to register before build (deleted on cleanup). |
| `pull_username`      | string    | —                | Ad-hoc registry username for `pull_image`. |
| `pull_password`      | string    | —                | Ad-hoc registry password for `pull_image`. |
| `pull_credential_id` | string    | —                | Saved tenant registry credential id (alternative to user/pass). |
| `pull_precache`      | bool      | `false`          | |
| `admin_username`     | string    | server-resolved  | Falls back to image label, then `admin`. |
| `ssh_key_ids`        | list      | —                | Existing tenant SSH key ids. When empty + ssh, an ephemeral key is generated. |
| `use_elastic_ip`     | bool      | `true`           | Allocate a public IP for SSH; otherwise use the private address. |
| `push_image`         | string    | —                | OCI reference to push the built VM to. |
| `push_username`      | string    | —                | |
| `push_password`      | string    | —                | |
| `push_credential_id` | string    | —                | Saved tenant registry credential id. |
| `push_precache`      | bool      | `false`          | |
| `keep_vm`            | bool      | `false`          | Skip teardown of the VM and temp resources. |
| `poll_interval`      | duration  | `5s`             | |
| `state_timeout`      | duration  | `20m`            | |
| `communicator`       | string    | `ssh`            | Only `ssh` or `none`. |

See [`example.pkr.hcl`](./example.pkr.hcl) for a complete macOS build.

## Build lifecycle (SSH)

1. **Register image** — register `pull_image` as a temporary catalog image
   (skipped when `image` is used).
2. **SSH key** — when `ssh_key_ids` is empty, generate an ephemeral ed25519
   keypair and register it.
3. **Create network** — optional (off by default; uses `network`).
4. **Create instance** — `POST /v1/xcloud/instances`.
5. **Wait running** — poll until `status == running` with no pending action.
6. **Resolve address** — poll the elastic IP until bound, or use the private
   `networkAddress`.
7. **Connect + provision** — Packer SSH communicator + provisioners.
8. **Shutdown + push** — graceful shutdown, then `push-image` job (only when
   `push_image` is set).

All created resources (instance, elastic IP, temp image/network, ephemeral
SSH key) are torn down on cleanup unless `keep_vm` is set.

## Out of scope

The REST API does not expose VNC keyboard automation, IPSW image import, or
API-key region discovery, so those upstream-plugin features are intentionally
absent here.

## Generate

The HCL2 spec is generated from the `Config` struct:

```bash
go generate ./...
```
