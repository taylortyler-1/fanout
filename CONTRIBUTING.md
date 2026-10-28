# Contributing to fanout

## Development

Use the Go version in `go.mod`, golangci-lint v2.11.4, and GoReleaser v2.

```sh
go build -o ./bin/fanout ./cmd/fanout
go test -race -cover ./...
go vet ./...
golangci-lint run ./...
golangci-lint fmt --diff
```

Tests cover skill discovery, validation, copying, manifests, output safety, and
the bundled Claude icon's format and dimensions. No Python tooling is required.
CI runs tests on Linux/macOS and lint on Linux.

Builtin platform scaffolds live in `internal/platform/platforms/` and are embedded
in the binary. See [custom platforms](docs/custom-platforms.md) for the template
format.

## Client smoke tests

Build a small trusted skill collection with metadata and both targets:

```sh
./bin/fanout build /path/to/test-skills -p all -o ./dist
claude plugin validate --strict ./dist/claude-code
claude --plugin-dir "$(pwd)/dist/claude-code"
pi -e "$(pwd)/dist/pi"
```

Confirm the skill is discoverable in each client. For installation testing, use
temporary client configuration directories rather than your personal settings.
Pi 1.4.0 and Claude Code 2.1.310 have been checked locally for installation and
skill discovery without model calls. The default icon passes Claude's listing
checks; publishing to its directory involves additional review.

## Releases

```sh
goreleaser check
goreleaser release --snapshot --clean
```

Snapshot archives are written to `dist/` without publishing. GoReleaser builds
with CGO disabled for Linux, macOS, and Windows on amd64 and arm64. Archives
include the README and MIT license, plus SHA-256 checksums. Windows binaries
are cross-compiled but have not yet been runtime-tested in Windows CI.
macOS binaries are not currently signed or notarized.

Before tagging, ensure the source is committed and pushed, CI passes, and
snapshot archives have been smoke-tested on the target systems. Check
installation and skill discovery in the supported agent clients.

```sh
git tag v0.1.0
git push origin v0.1.0
```

Use the next appropriate semantic version for subsequent releases.
`.github/workflows/release.yml` reruns CI before GoReleaser publishes the
GitHub release and its assets. Creating the GitHub repository alone does not
publish binaries; the release workflow runs when a version tag is pushed.
