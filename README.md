# mc-fleet

[![release](https://img.shields.io/github/v/release/meappy/mc-fleet)](https://github.com/meappy/mc-fleet/releases)
[![go.mod](https://img.shields.io/github/go-mod/go-version/meappy/mc-fleet)](go.mod)

Cross-platform single-binary launcher that syncs your Minecraft mods against
a published spec and starts the game. No Python, no JVM, no terminal pop-up
on macOS or Windows.

Built for the [meappy](https://minecraft.meappy.me) server, designed to work
for anyone running their own modded fleet.

## Install

### macOS / Linux

```sh
curl -fsSL https://github.com/meappy/mc-fleet/releases/latest/download/install.sh | sh
```

(Drops `meappy-mc` into `~/.local/bin/`.)

### Windows

Grab the `.exe` from the
[latest release](https://github.com/meappy/mc-fleet/releases/latest):

- `meappy-mc-windows-amd64.exe` (Intel/AMD)

Double-click it or run from the terminal.

### From source

```sh
go install github.com/meappy/mc-fleet/cmd/mc-fleet@latest
go install github.com/meappy/mc-fleet/cmd/meappy-mc@latest
```

## Two binaries — what's the difference

| | Binary | Use case |
|---|---|---|
| `meappy-mc` | brand-locked wrapper, hits the [meappy spec URL](https://minecraft.meappy.me/fleet-spec.json) | meappy players double-click this |
| `mc-fleet`  | generic CLI: `mc-fleet sync --remote URL`, `mc-fleet launch`, `mc-fleet apply` | run your own fleet — point `--remote` anywhere |

## How it works

```
1. Fetch fleet-spec.json from the public URL
2. For each profile in the spec:
   a. Resolve every Modrinth slug + URL entry
   b. Compare against on-disk .<slug>.version sentinels
   c. Download + SHA-512 verify only the jars that changed
3. Spawn Minecraft.app / Minecraft.exe / minecraft-launcher
```

Idempotent — running it again with everything up to date is a few HTTP
HEADs and a process spawn.

## Spec format

Same `fleet-spec.json` schema documented in the spec file itself; example:

```json
{
  "schema_version": 1,
  "servers": {
    "dragons": {
      "minecraft_version": "1.21.1",
      "loader": "neoforge",
      "address": "minecraft.meappy.me:25566",
      "mods": [
        "dmr",
        "jei:beta",
        "sophisticated-backpacks:1.21.1-3.25.53.1822",
        { "slug": "mouse-tweaks", "side": "client" },
        { "slug": "nfm-patched", "url": "https://example/nfm.jar" }
      ]
    }
  }
}
```

## Build + release

Tag a version and GoReleaser builds 7 cross-platform binaries + publishes
to GitHub Releases:

```sh
git tag v0.1.0
git push origin v0.1.0
```

GH Actions (`.github/workflows/release.yml`) handles the rest.

## License

MIT — see [LICENSE](LICENSE).
