# Changelog

All notable changes to NexaCLI are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and NexaCLI adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [2.4.1] — 2026-11-05

### Fixed
- Resolved a PATH update regression on zsh in macOS Sonoma (15.x) where the
  installer appended to `~/.zshrc` instead of `~/.zprofile`, causing `nexacli`
  to be unavailable in login shells. (#312)
- `nexacli rollback --region` now returns a clear error message when the
  specified region is not present in `nexacli.yaml` rather than panicking. (#308)
- Fixed a race condition in the health-check poller that could cause spurious
  "deployment failed" errors under high latency to the provider API. (#305)
- Corrected Windows binary detection in the install script; `install.ps1` no
  longer falls back to the Linux binary on Windows ARM64 machines. (#299)

### Performance
- Reduced peak memory usage during large image builds by streaming layer
  metadata rather than buffering it in memory. Saves ~40 MB on 2 GB+ images.
- Build context upload to AWS ECR and GCP Artifact Registry is now streamed
  with gzip compression enabled by default, cutting upload time by ~35% on
  typical Node.js projects. (#302)

---

## [2.4.0] — 2026-10-14

### Added
- **Multi-region rollbacks**: `nexacli rollback` now supports rolling back
  individual regions with `--region <name>`. When omitted, all regions defined
  in `nexacli.yaml` are rolled back in parallel. (#278)
- **Fly.io provider**: added first-class support for deploying to Fly.io
  (`provider: fly`). Supports regions, scaling, and health checks via the
  Fly Machines API. (#271)
- **Railway provider**: added support for Railway deployments
  (`provider: railway`). Env vars set via `nexacli env set` are synced to
  Railway's variable store before each deploy. (#265)
- **Render provider**: added support for Render web services
  (`provider: render`). (#269)
- `nexacli deploy --dry-run`: prints the resolved deployment plan without
  making any API calls. Useful for CI validation. (#261)
- `nexacli.yaml` now supports a top-level `labels` map; labels are propagated
  to provider-native tagging (AWS resource tags, GCP labels, Azure tags). (#258)

### Changed
- The `scaling` block is now honoured on GCP Cloud Run (min-instances and
  max-instances). Previously it was silently ignored. (#273)
- Default health-check path changed from `/` to `/health` in `nexacli init`
  generated configs. Existing configs are unaffected. (#260)

### Fixed
- `nexacli env set` no longer truncates the `.nexacli-env` file when writing
  fails mid-way; writes are now atomic (write to temp + rename). (#255)
- Resolved an issue where `nexacli deploy` would succeed even when the
  container image push timed out, leading to a stale deployment. (#252)

---

## [2.3.2] — 2026-07-22

### Fixed
- Fixed broken pipe error when piping `nexacli deploy` output through `tee`
  on Linux. (#241)
- Corrected region name validation to allow GCP multi-region identifiers
  such as `us` and `eu`. (#237)

---

## [2.3.1] — 2026-06-03

### Fixed
- `nexacli init` now correctly detects existing `nexacli.yaml` files even
  when the working directory contains symlinks. (#229)
- Improved error message when Docker daemon is not running:
  *"cannot connect to Docker daemon — is Docker running?"* (#225)

### Docs
- Updated CI/CD integration examples to use GitHub Actions `ubuntu-24.04`
  runner image. (#222)

---

## [2.3.0] — 2026-04-18

### Added
- **`env` subcommand**: `nexacli env set KEY=VALUE` and `nexacli env get KEY`
  for managing per-project environment variables stored in `.nexacli-env`. (#208)
- Azure provider (`provider: azure`) now supports deployment to Azure
  Container Apps in addition to App Service. Configure with
  `azure_target: container_apps` in `nexacli.yaml`. (#203)
- `nexacli --version` now also prints the Go runtime version and OS/arch. (#197)

### Changed
- Minimum Go version bumped from 1.20 to 1.21 to take advantage of
  `slices` and `maps` standard library packages. (#194)

### Deprecated
- The top-level `region` (singular) field in `nexacli.yaml` is deprecated in
  favour of the `regions` list added in v2.2.0. It will be removed in v3.0.0.
  A deprecation warning is printed at deploy time. (#199)

---

## [2.2.1] — 2025-12-10

### Fixed
- Installer `install.sh` no longer fails on Alpine Linux where `id -u`
  returns a string rather than an integer. (#185)
- Fixed incorrect registry URL construction for GCP Artifact Registry when
  the project ID contains uppercase letters. GCP project IDs are lowercased
  before use. (#181)

---

## [2.2.0] — 2025-10-02

### Added
- Multi-region configuration via the `regions` list in `nexacli.yaml`.
  Deployments are now rolled out region-by-region, primary first. (#162)
- `nexacli rollback` supports targeting a specific previous deploy ID with
  `--deploy-id <id>`. (#158)

### Fixed
- Health-check retries are now correctly bounded; previously a misconfigured
  `retries: 0` would spin indefinitely. (#155)

---

## [2.1.0] — 2025-07-14

### Added
- GCP Cloud Run support (`provider: gcp`). (#134)
- Autoscaling configuration block (`scaling.min`, `scaling.max`,
  `scaling.target`). (#130)

### Fixed
- `nexacli init` now writes `nexacli.yaml` with `0644` permissions rather than
  `0666`, respecting the user's umask. (#127)

---

## [2.0.0] — 2025-04-01

### Breaking Changes
- Configuration file renamed from `nexa.yml` to `nexacli.yaml`. Run
  `nexacli migrate` (new in this release) to rename automatically. (#100)
- `nexacli deploy --env` flag removed; use the `env` block in `nexacli.yaml`
  or the `nexacli env set` subcommand (added in v2.3.0). (#98)

### Added
- AWS ECS / Fargate provider (`provider: aws`). (#94)
- Azure App Service provider (`provider: azure`). (#97)
- Build args support via `build.args` in `nexacli.yaml`. (#91)

---

## [1.x]

Earlier 1.x releases predating the multi-provider rewrite are archived at
<https://github.com/adahir-gif/NexaCLI/releases?q=1.&expanded=true>.
