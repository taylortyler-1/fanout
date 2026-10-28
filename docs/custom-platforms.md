# Custom platform templates

A platform is a directory with a `platform.yaml` descriptor and a `files/` tree:

```text
my-platform/
  platform.yaml
  files/
    manifest.json.tmpl
    LICENSE
```

For example, `platform.yaml` might contain:

```yaml
name: my-platform
description: Package skills for another agent client
skillsDir: skills
```

The name must be kebab-case. `skillsDir` is a relative path inside the generated
plugin and defaults to `skills`.

Build with:

```sh
fanout build ./my-skills --platform-dir ./my-platform
```

Files ending in `.tmpl` are rendered using Go's `text/template`, with the suffix
removed. Other files are copied as-is. Rendered `.json` files must be valid JSON.
The output is written to `<out>/<platform-name>`.

Templates receive:

| Value | Contents |
| --- | --- |
| `.Plugin` | Plugin metadata from `fanout.yaml` and flags |
| `.Skills` | Selected skills, including `.Name`, `.Description`, `.License`, `.Metadata` |
| `.Platform` | The platform descriptor |
| `.SkillsDir` | The destination directory for skills |

The `json` function serializes a value safely for JSON. `join` joins strings.
For example:

```gotemplate
{
  "name": {{ json .Plugin.Name }},
  "version": {{ json .Plugin.Version }}
}
```

Selected skill directories are copied into `<skillsDir>/<skill-name>` after the
scaffold is rendered. Custom templates are trusted input, not a sandbox.

For complete examples, see the
[builtin platforms](../internal/platform/platforms). To customize Claude output
persistently, copy its platform directory, replace
`files/.claude-plugin/icon.png` with your own icon, and build using
`--platform-dir` instead of selecting the builtin `claude-code` target.
