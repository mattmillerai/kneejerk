# Contributing

## Commit / PR title format

Releases are automated with [release-please](https://github.com/googleapis/release-please), which parses
[Conventional Commits](https://www.conventionalcommits.org/) to decide the next version and changelog entry.
PRs are squash-merged, so **the PR title becomes the commit message on `main`** — it must follow the
Conventional Commits format:

```
feat: extract hostnames from inline sourcemaps
fix: don't panic on empty response body
deps: bump golang.org/x/net to v0.61.0
chore: tidy up helper tests
ci: cache go modules in CI
docs: document the -domain flag
feat!: drop support for the legacy -json flag
```

A `!` after the type (or a `BREAKING CHANGE:` footer in the body) marks a breaking change.

## How commit types map to version bumps

kneejerk is pre-1.0, and `release-please-config.json` sets `bump-minor-pre-major: true` and
`bump-patch-for-minor-pre-major: false`. That means, until we cut 1.0.0:

* `fix:` → patch bump (`0.3.1` → `0.3.2`)
* `feat:` → minor bump (`0.3.1` → `0.4.0`)
* `feat!:` / `BREAKING CHANGE:` → **minor** bump, not major (breaking changes are expected pre-1.0)
* `deps:` → patch bump, listed under "Dependencies" (Dependabot Go module PRs use this prefix, so dependency/security bumps ship as patch releases)
* `chore:`, `ci:`, `docs:`, `test:` → hidden from the changelog and don't bump the version

## Release flow

1. Every qualifying commit on `main` is picked up by release-please, which keeps a single open
   "Release PR" with the version bump and `CHANGELOG.md` entry up to date.
2. Merging that PR tags `vX.Y.Z` and publishes the GitHub release.
3. GoReleaser then builds and attaches `linux`/`darwin` `amd64`/`arm64` binaries plus a `checksums.txt`
   to that release.
4. The Homebrew formula in [mattmillerai/homebrew-tap](https://github.com/mattmillerai/homebrew-tap) is
   bumped to the new tag automatically.

All of this is driven by `.github/workflows/release.yml` — no manual tagging or publishing needed.
