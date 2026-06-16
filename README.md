# projectdetect

[![CI](https://github.com/richardwooding/projectdetect/actions/workflows/ci.yml/badge.svg)](https://github.com/richardwooding/projectdetect/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/richardwooding/projectdetect.svg)](https://pkg.go.dev/github.com/richardwooding/projectdetect)

Detect what kind of project a directory is — pure Go, no cgo.

`projectdetect` answers two questions over a filesystem:

- **`Detect(fsys, dir)`** — what project type(s) does *this* directory look like? (a directory can match several at once — a Go module that also ships a `docker-compose.yml` matches both)
- **`Find(ctx, root, opts)`** — walk a tree and report every project root under it.

A type matches by **indicators**: an exact filename (`HasFile`), a file-basename glob (`HasGlob`), a subdirectory-basename glob (`HasSubdirGlob`, for directory markers like `*.xcodeproj`), or — optionally — a CEL expression over the directory's `files` / `subdirs`.

## Built-in types

`go`, `node`, `rust`, `python`, `ruby`, `java-maven`, `java-gradle`, `dotnet`, `terraform`, `docker-compose`, the language / build-tool ecosystems `swift`, `php`, `scala-sbt`, `scala-mill`, `cmake`, `autotools`, `r`, `zig`, `perl`, `matlab`, and the static-site generators `hugo`, `jekyll`, `eleventy`, `astro`, `gatsby`, `mkdocs`, `docusaurus`, `pelican` (28 total). The `dotnet` type covers `*.csproj` / `*.fsproj` / `*.vbproj` / `*.sln` / `*.slnx` plus `global.json` / `Directory.Build.props` / `Directory.Packages.props` / `nuget.config`. `swift` matches `Package.swift` (SwiftPM), `*.podspec` (CocoaPods), and the `*.xcodeproj` / `*.xcworkspace` bundles (Xcode); `cmake` matches `CMakeLists.txt` (C/C++).

Each type also declares its canonical build-artefact dirs (`bin`/`obj`, `node_modules`, `target`, …) — see `CollectBuildExcludes`.

## Install

```sh
go get github.com/richardwooding/projectdetect
```

## Usage

```go
package main

import (
	"fmt"
	"os"

	"github.com/richardwooding/projectdetect"
)

func main() {
	for _, m := range projectdetect.Detect(os.DirFS("."), ".") {
		fmt.Printf("%s (via %s)\n", m.Type, m.Indicator)
	}
}
```

Recursively find project roots under a tree:

```go
res, err := projectdetect.Find(ctx, "/path/to/code", projectdetect.FindOptions{})
```

## Custom types (YAML)

Load extra project types from YAML — `has_file`, `has_glob`, `has_subdir_glob`, or `cel`:

```yaml
project_types:
  - name: my-stack
    indicators:
      - has_file: "my.config"
      - has_glob: "*.mytool"
      - has_subdir_glob: "*.bundle"
      - cel: '"services" in subdirs && "compose.yaml" in files'
```

```go
n, err := projectdetect.LoadFromFile("project-types.yaml") // registers into the default registry
```

## CEL indicators are opt-in (no cel-go unless you want it)

The base package has **no CEL dependency**. `cel:` indicators are compiled by the
`celindicators` sub-package — enable them with a blank import:

```go
import _ "github.com/richardwooding/projectdetect/celindicators"
```

Without it, `HasFile` / `HasGlob` indicators and all built-ins work as normal;
registering a type that uses a `cel:` indicator returns a clear error telling you
to add the import. This keeps `cel-go` (and its transitive deps) out of the build
for consumers that only need filename/glob matching.

## Provenance

Extracted from [`file-search-on`](https://github.com/richardwooding/file-search-on),
where it powers the `detect-project` / `find-projects` / `which-project` commands.

## License

MIT — see [LICENSE](LICENSE).
