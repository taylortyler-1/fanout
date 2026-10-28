# fanout

Build Claude Code plugins and Pi packages from the same collection of
[Agent Skills](https://agentskills.io). Write your skills once; fanout validates
them, copies their files, and adds the manifests each client needs.

```sh
fanout build ./my-skills -p all
```

## Install

### Prebuilt binaries — no Go required

Download the archive for your system and `checksums.txt` from
[GitHub Releases](https://github.com/taylortyler-1/fanout/releases).

| System | Archive platform | Architecture |
| --- | --- | --- |
| macOS, Apple Silicon | `darwin` | `arm64` |
| macOS, Intel | `darwin` | `amd64` |
| Linux | `linux` | `amd64` or `arm64` |
| Windows | `windows` | `amd64` or `arm64` |

1. Verify the archive's SHA-256 against `checksums.txt` (`shasum -a 256` on
   macOS, `sha256sum` on Linux, or `Get-FileHash` in PowerShell).
2. Extract the `.tar.gz` (macOS/Linux) or `.zip` (Windows).
3. Put `fanout` or `fanout.exe` in a directory on your `PATH`.
4. Run `fanout --version` to check the installation.

All platform templates are included in the binary. macOS binaries are not yet
signed or notarized.

### From source

Requires the Go version specified in [go.mod](go.mod). From a source checkout:

```sh
go build -o ./bin/fanout ./cmd/fanout
./bin/fanout --help
```

Put the built binary on your `PATH` to use the commands below.

## Quick start

Start with a directory containing one or more skills:

```text
my-skills/
  fanout.yaml
  explain-code/
    SKILL.md
    references/       # optional supporting files
```

Each skill needs YAML frontmatter in its `SKILL.md`:

```markdown
---
name: explain-code
description: Explain how code works. Use when walking through unfamiliar code.
---

Explain the code step by step, including important assumptions and edge cases.
```

The name must match its directory. Add `fanout.yaml` at the collection root:

```yaml
name: my-skills
version: 0.1.0
description: Example skills
author:
  name: Your Name
license: MIT
```

Use your own author details and the appropriate license for your skills.

Then validate and build:

```sh
fanout validate ./my-skills
fanout build ./my-skills -p all
```

Output is written relative to your current directory:

```text
dist/
  claude-code/        # plugin + single-plugin marketplace + default fan icon
  pi/                 # package.json + explicit Pi skills manifest
```

Your skill files are copied unchanged into each output's `skills/` directory.
Without `-p all`, fanout uses the platforms in `fanout.yaml`, or `claude-code`
if none are configured.

## Use your skills

### Claude Code

Try the plugin for one session:

```sh
claude --plugin-dir "$(pwd)/dist/claude-code"
```

Or install it through its bundled marketplace:

```sh
claude plugin marketplace add "$(pwd)/dist/claude-code"
claude plugin install my-skills@my-skills
```

Replace `my-skills` with the `name` in your config. Invoke the example skill as
`/my-skills:explain-code`.

**Default icon:** fanout includes a 512×512 fan icon in every Claude Code build
at `.claude-plugin/icon.png`, so you do not need to supply an icon to pass the
listing-icon check. You can replace that file in the generated output with your
own square PNG (512–2048 px, under 2 MB). Do this before your first directory
submission if you want your own branding. Rebuilding restores the default icon
along with the rest of the output, so reapply any customization after rebuilding.

Validate the generated plugin with Claude Code:

```sh
claude plugin validate --strict ./dist/claude-code
```

### Pi

Try the package for one session:

```sh
pi -e "$(pwd)/dist/pi"
```

Or install it:

```sh
pi install "$(pwd)/dist/pi"
```

Local Pi packages are loaded in place, not copied: keep the output directory
after installation. Restart Pi or run `/reload`, then use `/skill:explain-code`.
The generated package can also be published to npm or a standalone Git repository;
its README includes distribution instructions.

## Configure your build

`fanout.yaml` is optional. Flags override its values.

```yaml
name: my-skills
version: 1.0.0
description: Skills for working with documents
author:
  name: Your Name
  email: you@example.com
  url: https://example.com
homepage: https://example.com
repository: https://github.com/you/my-skills
license: MIT
keywords: [docs]
platforms: [claude-code, pi]   # default: [claude-code]
skills: [explain-code]         # default: all discovered skills
exclude: [dist, examples]      # default: [dist]
```

Without a config, the plugin name comes from the repository directory and its
version defaults to `0.1.0`. Set a description and author to avoid Claude's
missing-metadata warnings. Claude's directory also requires a license.
`license` describes your packaged skills, not fanout itself.

### Common commands

```sh
fanout platforms                               # list builtin targets
fanout build ./my-skills -p pi                  # Pi only
fanout build ./my-skills -p all -o packages      # choose output directory
fanout build ./my-skills -p all --force          # replace existing output
fanout build ./my-skills --skill explain-code   # include one skill
fanout build ./my-skills --name team-skills --plugin-version 1.2.0
fanout build ./my-skills --config ./other.yaml
```

Use `fanout build --help` for all options. For another agent client, see
[custom platform templates](docs/custom-platforms.md).

## Files and safety

- Hidden directories, `node_modules`, and excluded directories are skipped when
  discovering skills. Skills cannot be nested inside another skill.
- Symlinks are followed and copied as real files, making output self-contained.
  Broken links inside a skill fail the build. Only build trusted repositories:
  symlinks can include files from outside the collection.
- Non-empty output directories require `--force`. Do not store source skills or
  files you want to keep in generated output.
- Rendering and copying complete before replacing each platform's previous
  output. A multi-platform build is not an all-or-nothing transaction.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development checks and release steps.

## License

[MIT](LICENSE). Packaging skills with fanout does not change their licenses.
