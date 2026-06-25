package projectdetect_test

import (
	"path/filepath"
	"testing"

	"github.com/richardwooding/projectdetect"
)

func TestResolver_FindsNearestProject(t *testing.T) {
	root := t.TempDir()
	// root/proj/go.mod  → 'proj' is a Go project
	// root/proj/cmd/main.go
	// root/proj/inner/Cargo.toml → 'inner' is a Rust project
	// root/proj/inner/src/lib.rs
	mustMkdir(t, filepath.Join(root, "proj", "cmd"))
	mustMkdir(t, filepath.Join(root, "proj", "inner", "src"))
	mustWrite(t, filepath.Join(root, "proj", "go.mod"), "module x\n")
	mustWrite(t, filepath.Join(root, "proj", "cmd", "main.go"), "package main\n")
	mustWrite(t, filepath.Join(root, "proj", "inner", "Cargo.toml"), "[package]\nname=\"x\"\n")
	mustWrite(t, filepath.Join(root, "proj", "inner", "src", "lib.rs"), "")

	resolver := projectdetect.NewResolver(root, nil)

	// main.go lives in proj/cmd → nearest ancestor is proj (go).
	matches := resolver.Resolve(filepath.Join(root, "proj", "cmd", "main.go"))
	if len(matches) != 1 || matches[0].Type != "go" {
		t.Errorf("cmd/main.go: matches=%+v, want [go]", matches)
	}

	// lib.rs lives in proj/inner/src → nearest ancestor is inner (rust).
	matches = resolver.Resolve(filepath.Join(root, "proj", "inner", "src", "lib.rs"))
	if len(matches) != 1 || matches[0].Type != "rust" {
		t.Errorf("inner/src/lib.rs: matches=%+v, want [rust] (closer than the outer Go project)", matches)
	}
}

func TestResolver_NoProject(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "loose.txt"), "")
	resolver := projectdetect.NewResolver(root, nil)
	if matches := resolver.Resolve(filepath.Join(root, "loose.txt")); matches != nil {
		t.Errorf("file outside any project should return nil, got %+v", matches)
	}
}

func TestResolveForPath_FindsNearest(t *testing.T) {
	root := t.TempDir()
	// root/proj/go.mod, root/proj/inner/Cargo.toml — nearest fires first.
	mustMkdir(t, filepath.Join(root, "proj", "cmd"))
	mustMkdir(t, filepath.Join(root, "proj", "inner", "src"))
	mustWrite(t, filepath.Join(root, "proj", "go.mod"), "module x\n")
	mustWrite(t, filepath.Join(root, "proj", "cmd", "main.go"), "package main\n")
	mustWrite(t, filepath.Join(root, "proj", "inner", "Cargo.toml"), "[package]\nname=\"x\"\n")
	mustWrite(t, filepath.Join(root, "proj", "inner", "src", "lib.rs"), "")

	gotRoot, matches := projectdetect.ResolveForPath(filepath.Join(root, "proj", "cmd", "main.go"), nil)
	if gotRoot != filepath.Join(root, "proj") {
		t.Errorf("root=%q want %q", gotRoot, filepath.Join(root, "proj"))
	}
	if len(matches) != 1 || matches[0].Type != "go" {
		t.Errorf("matches=%+v want [go]", matches)
	}

	// Nested project — inner Cargo.toml wins over outer go.mod.
	gotRoot, matches = projectdetect.ResolveForPath(filepath.Join(root, "proj", "inner", "src", "lib.rs"), nil)
	if gotRoot != filepath.Join(root, "proj", "inner") {
		t.Errorf("nested root=%q want %q", gotRoot, filepath.Join(root, "proj", "inner"))
	}
	if len(matches) != 1 || matches[0].Type != "rust" {
		t.Errorf("nested matches=%+v want [rust]", matches)
	}
}

func TestResolveForPath_NoProject(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "loose.txt")
	mustWrite(t, p, "")
	gotRoot, matches := projectdetect.ResolveForPath(p, nil)
	if gotRoot != "" || matches != nil {
		t.Errorf("no-project lookup: root=%q matches=%+v; want (empty, nil)", gotRoot, matches)
	}
}

func TestResolveForPath_PolyglotMultipleTypes(t *testing.T) {
	// A directory that fires multiple types (Go + docker-compose) —
	// both should appear in the matches list.
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "go.mod"), "module x\n")
	mustWrite(t, filepath.Join(root, "docker-compose.yml"), "services: {}\n")
	mustMkdir(t, filepath.Join(root, "cmd"))
	mustWrite(t, filepath.Join(root, "cmd", "main.go"), "package main\n")

	gotRoot, matches := projectdetect.ResolveForPath(filepath.Join(root, "cmd", "main.go"), nil)
	if gotRoot != root {
		t.Errorf("root=%q want %q", gotRoot, root)
	}
	if len(matches) < 2 {
		t.Fatalf("polyglot matches=%+v want >=2 (go and docker-compose)", matches)
	}
	gotTypes := map[string]bool{}
	for _, m := range matches {
		gotTypes[m.Type] = true
	}
	if !gotTypes["go"] || !gotTypes["docker-compose"] {
		t.Errorf("polyglot matches=%+v missing go and/or docker-compose", matches)
	}
}

func TestResolver_SkipsVCSDirsByDefault(t *testing.T) {
	// A file inside .git must resolve to the enclosing project, NOT to
	// the .git dir (even when .git contortedly contains a marker).
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "go.mod"), "module x\n")
	mustMkdir(t, filepath.Join(root, ".git"))
	mustWrite(t, filepath.Join(root, ".git", "go.mod"), "module ghost\n")

	resolver := projectdetect.NewResolver(root, nil)
	matches := resolver.Resolve(filepath.Join(root, ".git", "config"))
	if len(matches) != 1 || matches[0].Type != "go" {
		t.Errorf(".git/config: matches=%+v, want [go] from the enclosing root (.git skipped)", matches)
	}

	// Opt back in: now .git itself resolves as a project.
	resolver = projectdetect.NewResolverWithOptions(root, nil, projectdetect.ResolveOptions{IncludeVCSDirs: true})
	matches = resolver.Resolve(filepath.Join(root, ".git", "config"))
	if len(matches) != 1 || matches[0].Type != "go" {
		t.Errorf(".git/config with IncludeVCSDirs: matches=%+v, want [go] (the .git dir's own marker)", matches)
	}
}

func TestResolver_SkipDirPredicate(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "go.mod"), "module outer\n")
	mustMkdir(t, filepath.Join(root, "skipme", "sub"))
	mustWrite(t, filepath.Join(root, "skipme", "Cargo.toml"), "[package]\nname=\"x\"\n")

	resolver := projectdetect.NewResolverWithOptions(root, nil, projectdetect.ResolveOptions{
		SkipDir: func(_, name string) bool { return name == "skipme" },
	})
	// A file under skipme/sub would normally resolve to skipme (rust),
	// but the predicate skips skipme, so the walk reaches the outer Go root.
	matches := resolver.Resolve(filepath.Join(root, "skipme", "sub", "file.txt"))
	if len(matches) != 1 || matches[0].Type != "go" {
		t.Errorf("matches=%+v, want [go] (skipme pruned, outer root wins)", matches)
	}
}

func TestResolveForPathWithOptions_SkipsVCS(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "go.mod"), "module x\n")
	mustMkdir(t, filepath.Join(root, ".git"))
	mustWrite(t, filepath.Join(root, ".git", "Cargo.toml"), "[package]\nname=\"x\"\n")

	gotRoot, matches := projectdetect.ResolveForPath(filepath.Join(root, ".git", "config"), nil)
	if gotRoot != root {
		t.Errorf("root=%q want %q (.git skipped as a candidate)", gotRoot, root)
	}
	if len(matches) != 1 || matches[0].Type != "go" {
		t.Errorf("matches=%+v want [go]", matches)
	}

	gotRoot, matches = projectdetect.ResolveForPathWithOptions(filepath.Join(root, ".git", "config"), nil, projectdetect.ResolveOptions{IncludeVCSDirs: true})
	if gotRoot != filepath.Join(root, ".git") || len(matches) != 1 || matches[0].Type != "rust" {
		t.Errorf("with IncludeVCSDirs: root=%q matches=%+v, want .git as rust", gotRoot, matches)
	}
}

func TestResolver_CustomRegistry(t *testing.T) {
	// Use an isolated registry with a single custom type so we can
	// verify the resolver targets the registry it was constructed
	// with, not defaultRegistry.
	reg := projectdetect.NewRegistry()
	if err := reg.Register(&projectdetect.ProjectType{
		Name: "custom",
		Indicators: []projectdetect.Indicator{
			{HasFile: "custom.marker"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "custom.marker"), "")
	mustWrite(t, filepath.Join(root, "go.mod"), "module x\n") // would fire 'go' in defaultRegistry
	mustWrite(t, filepath.Join(root, "file.txt"), "")

	resolver := projectdetect.NewResolver(root, reg)
	matches := resolver.Resolve(filepath.Join(root, "file.txt"))
	if len(matches) != 1 || matches[0].Type != "custom" {
		t.Errorf("custom resolver: matches=%+v, want [custom] (NOT [go] from defaultRegistry)", matches)
	}
}
