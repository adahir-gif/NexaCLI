# NexaCLI

[![Go version](https://img.shields.io/badge/go-1.21-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)
[![Latest release](https://img.shields.io/github/v/release/adahir-gif/NexaCLI)](https://github.com/adahir-gif/NexaCLI/releases/latest)
[![CI](https://github.com/adahir-gif/NexaCLI/actions/workflows/release.yml/badge.svg)](https://github.com/adahir-gif/NexaCLI/actions)

**NexaCLI** is a zero-config CLI tool for deploying containerised applications
to any major cloud provider from your terminal. Write one `nexacli.yaml`, run
`nexacli deploy` — NexaCLI handles the rest.

```
$ nexacli deploy
Deploying my-api  (provider: aws, deploy-id: dep-1730749200)

  Authenticating with provider...           done
  Resolving build context...                done
  Building container image...               done
  Pushing image to registry...              done
  Updating deployment manifest...           done
  Waiting for rollout to complete...        done
  Running health checks...                  done

Deployment successful! App "my-api" is live.
  https://my-api.us-east-1.nexacli.app [primary]
  https://my-api.eu-west-1.nexacli.app
```

---

## Features

- **One config, six providers** — AWS ECS/Fargate, GCP Cloud Run, Azure
  Container Apps, Fly.io, Railway, and Render all share the same `nexacli.yaml`
  schema.
- **Multi-region rollouts** — declare multiple regions; NexaCLI deploys them
  sequentially, primary region first.
- **Safe rollbacks** — `nexacli rollback` reinstates the previous stable image
  across all regions, or just the one you specify with `--region`.
- **Environment variable management** — `nexacli env set/get` stores key-value
  pairs in a local `.nexacli-env` file (plaintext; add it to `.gitignore`) and
  passes them to the provider over TLS on each deploy.
- **Dry-run mode** — `nexacli deploy --dry-run` prints the resolved plan
  without touching a single API.
- **CI-friendly** — structured exit codes, no interactive prompts, and a
  ready-made GitHub Actions workflow in [`.github/workflows/release.yml`](.github/workflows/release.yml).

---

## Installation

### Linux / macOS

```bash
curl -fsSL https://nexacli-production.up.railway.app/install.sh | bash
```

The installer places the `nexacli` binary in `/usr/local/bin` (or `~/.local/bin`
when run without root) and adds it to your `PATH`.

### Windows (PowerShell)

```powershell
iwr -useb https://nexacli-production.up.railway.app/install.ps1 | iex
```

The installer places `nexacli.exe` in `$env:USERPROFILE\.local\bin` and adds it
to your user `PATH`.

### From source

```bash
go install github.com/adahir-gif/NexaCLI/cmd/nexacli@latest
```

Requires Go 1.21 or later.

### Pre-built binaries

Download the latest release from the
[Releases page](https://github.com/adahir-gif/NexaCLI/releases/latest).
Binaries are provided for Linux (amd64, arm64), macOS (amd64, arm64), and
Windows (amd64), with SHA-256 checksums in `checksums.txt`.

---

## Quick start

```bash
# 1. Initialise a config in your project root
cd my-app
nexacli init --provider aws

# 2. Edit nexacli.yaml to taste, then deploy
nexacli deploy

# 3. Something went wrong? Roll back instantly
nexacli rollback
```

---

## Command reference

| Command | Description |
|---|---|
| `nexacli init` | Create `nexacli.yaml` in the current directory |
| `nexacli init --provider <p>` | Specify provider (`aws`\|`gcp`\|`azure`\|`fly`\|`railway`\|`render`) |
| `nexacli deploy` | Build image and deploy to all configured regions |
| `nexacli deploy --dry-run` | Print what would happen without calling any APIs |
| `nexacli rollback` | Roll back to the previous stable version across all regions |
| `nexacli rollback --region <r>` | Roll back only the named region |
| `nexacli env set KEY=VALUE` | Persist an environment variable to `.nexacli-env` |
| `nexacli env get KEY` | Read a variable from `.nexacli-env` |
| `nexacli --version` | Print version and exit |
| `nexacli help` | Print usage information |

---

## Configuration

NexaCLI is configured via `nexacli.yaml` in your project root.  
See [`nexacli.yaml.example`](nexacli.yaml.example) for a fully-annotated template.

```yaml
version: "1"
app: my-api
provider: aws

regions:
  - name: us-east-1
    primary: true
  - name: eu-west-1
    replicas: 2

build:
  dockerfile: Dockerfile
  context: .
  args:
    NODE_ENV: production

healthcheck:
  path: /health
  interval: 30
  timeout: 10
  retries: 3

scaling:
  min: 1
  max: 5
  target: 70

env:
  LOG_LEVEL: info
  PORT: "8080"

labels:
  team: platform
```

### Schema reference

| Field | Type | Default | Description |
|---|---|---|---|
| `version` | string | — | Config schema version; always `"1"` for NexaCLI 2.x |
| `app` | string | — | Application name **(required)** |
| `provider` | string | — | Cloud provider **(required)** |
| `regions[].name` | string | — | Provider region identifier |
| `regions[].primary` | bool | `false` | First-deployed region; must have exactly one |
| `regions[].replicas` | int | `1` | Number of running instances in this region |
| `build.dockerfile` | string | `Dockerfile` | Path to Dockerfile (relative to context) |
| `build.context` | string | `.` | Docker build context |
| `build.args` | map | — | `--build-arg` values passed to `docker build` |
| `healthcheck.path` | string | `/health` | HTTP path checked after each deploy |
| `healthcheck.interval` | int | `30` | Seconds between health checks |
| `healthcheck.timeout` | int | `10` | Seconds before a check times out |
| `healthcheck.retries` | int | `3` | Failures before the deployment is considered failed |
| `scaling.min` | int | `1` | Minimum running instances |
| `scaling.max` | int | `3` | Maximum running instances |
| `scaling.target` | int | `70` | CPU % that triggers a scale-up |
| `env` | map | — | Static environment variables baked into the deployment |
| `labels` | map | — | Arbitrary key-value labels propagated to provider tagging |

---

## CI/CD integration

### GitHub Actions

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install NexaCLI
        run: curl -fsSL https://nexacli-production.up.railway.app/install.sh | bash

      - name: Deploy
        env:
          NEXACLI_TOKEN: ${{ secrets.NEXACLI_TOKEN }}
        run: nexacli deploy
```

### GitLab CI

```yaml
# .gitlab-ci.yml
deploy:
  stage: deploy
  image: ubuntu:24.04
  only:
    - main
  script:
    - curl -fsSL https://nexacli-production.up.railway.app/install.sh | bash
    - nexacli deploy
  variables:
    NEXACLI_TOKEN: $NEXACLI_TOKEN
```

---

## Contributing

Contributions are welcome! Please open an issue before submitting a large PR so
we can discuss the approach.

```bash
# Clone the repo
git clone https://github.com/adahir-gif/NexaCLI.git
cd NexaCLI

# Install dependencies
go mod download

# Build
go build ./cmd/nexacli

# Run tests
go test ./...
```

Please follow the existing code style, include tests for new behaviour, and
update `CHANGELOG.md` under the `[Unreleased]` heading.

---

## License

Copyright 2024 NexaLabs Inc.  
Licensed under the [Apache License 2.0](LICENSE).
