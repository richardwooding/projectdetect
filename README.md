# projectdetect

[![CI](https://github.com/richardwooding/projectdetect/actions/workflows/ci.yml/badge.svg)](https://github.com/richardwooding/projectdetect/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/richardwooding/projectdetect.svg)](https://pkg.go.dev/github.com/richardwooding/projectdetect)

**Website:** [richardwooding.github.io/projectdetect](https://richardwooding.github.io/projectdetect/)

Detect what kind of project a directory is — pure Go, no cgo.

`projectdetect` answers two questions over a filesystem:

- **`Detect(fsys, dir)`** — what project type(s) does *this* directory look like? (a directory can match several at once — a Go module that also ships a `docker-compose.yml` matches both)
- **`Find(ctx, root, opts)`** — walk a tree and report every project root under it.

A type matches by **indicators**: an exact filename (`HasFile`), a file-basename glob (`HasGlob`), a subdirectory-basename glob (`HasSubdirGlob`, for directory markers like `*.xcodeproj`), or — optionally — a CEL expression over the directory's `files` / `subdirs`.

## Built-in types

`go`, `node`, `rust`, `python`, `ruby`, `java-maven`, `java-gradle`, `dotnet`, `terraform`, `docker-compose`, the language / build-tool ecosystems `swift`, `php`, `scala-sbt`, `scala-mill`, `cmake`, `autotools`, `r`, `zig`, `perl`, `matlab`, and the static-site generators `hugo`, `jekyll`, `eleventy`, `astro`, `gatsby`, `mkdocs`, `docusaurus`, `pelican` (28 total). The `dotnet` type covers `*.csproj` / `*.fsproj` / `*.vbproj` / `*.sln` / `*.slnx` plus `global.json` / `Directory.Build.props` / `Directory.Packages.props` / `nuget.config`. `swift` matches `Package.swift` (SwiftPM), `*.podspec` (CocoaPods), and the `*.xcodeproj` / `*.xcworkspace` bundles (Xcode); `cmake` matches `CMakeLists.txt` (C/C++).

Each type also declares its canonical build-artefact dirs (`bin`/`obj`, `node_modules`, `target`, …) — see `CollectBuildExcludes`.

## Detect a file's language

A separate axis from project types: project types answer *"what kind of project is this directory?"*, while `LanguageForPath` answers *"what language is this single file?"* by its extension. A language commonly has several extensions (C++ alone has a dozen header/source spellings), so the mapping is one-to-many.

```go
projectdetect.LanguageForPath("src/app.py")   // "python"
projectdetect.LanguageForPath("Widget.cpp")   // "cpp"
projectdetect.LanguageForExt(".rs")           // "rust"  (".rs", "rs", ".RS" all work)
projectdetect.LanguageForPath("README.md")    // ""      (unrecognised)

projectdetect.Languages()                      // sorted ids
projectdetect.ExtensionsForLanguage("python")  // [".pxd" ".py" ".pyi" ".pyw" ".pyx"]
```

Recognised ids: `go`, `python`, `javascript`, `typescript`, `java`, `rust`, `c`, `cpp`, `csharp`, `kotlin`, `php`, `ruby`, `scala`, `r`, `matlab`, `perl`, `swift`. A few ambiguous extensions are assigned to a single language by convention — `.h`→`c`, `.m`→`matlab`, `.sc`→`scala`, `.t`→`perl` — so the extension→language map is 1:1.

## Skip vendored / minified files

A third axis: *"is this file third-party content an analysis tool should skip?"* Unlike a project type's `BuildExcludes` (which prune whole directories), this classifies individual **file paths** — so a bundled `jquery.min.js` or a vendored library committed anywhere (even under `docs/`) is caught. The default patterns are a curated subset of [GitHub Linguist's `vendor.yml`](https://github.com/github-linguist/linguist/blob/main/lib/linguist/vendor.yml).

```go
projectdetect.IsVendored("docs/js/jquery.min.js")     // true (minified)
projectdetect.IsVendored("node_modules/x/index.js")   // true (dependency dir)
projectdetect.IsVendored("website/scripts/prism.js")  // true (known library)
projectdetect.IsVendored("src/main.go")               // false (real source)
```

**Flexible** — extend the built-ins, or build your own set:

```go
// Add patterns to the process-wide default used by IsVendored:
projectdetect.RegisterVendorPattern(`\.pb\.go$`, `(^|/)generated/`)

// Or a fully custom matcher (regexps over the slash-separated path):
m := projectdetect.NewVendorMatcher()          // empty; DefaultVendorMatcher() preloads built-ins
_ = m.Add(`(^|/)my-vendor/`)
m.Match("libs/my-vendor/thing.js")             // true
```

For bundles with no telltale name, `IsMinified(content []byte)` judges by shape (a meaningfully-sized file whose average line length far exceeds normal source):

```go
projectdetect.IsMinified(data) // true for a packed one-liner, false for hand-written source
```

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

## Pruning the walk

`Find`, `CollectBuildExcludes`, and the resolver walk-up all prune
version-control metadata directories (`.git`, `.hg`, `.svn`) by default —
they never hold project roots and walking them is wasted I/O. Set
`IncludeVCSDirs: true` to descend into them.

To mirror an external walker's exclusions exactly, pass a `SkipDir`
predicate. It's consulted for every directory below the walk root;
returning `true` prunes that whole subtree:

```go
res, err := projectdetect.Find(ctx, root, projectdetect.FindOptions{
    SkipDir: func(relPath, name string) bool {
        return name == "node_modules" || name == "target"
    },
})

// Same hook on the build-artefact collector …
ex, err := projectdetect.CollectBuildExcludesWithOptions(ctx, root, projectdetect.FindOptions{
    SkipDir: skip,
})

// … and on the per-file resolver walk-up.
r := projectdetect.NewResolverWithOptions(root, nil, projectdetect.ResolveOptions{SkipDir: skip})
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
