package projectdetect_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/richardwooding/projectdetect"
)

func TestDetect_SingleType(t *testing.T) {
	cases := []struct {
		name     string
		indicator string
		wantType string
	}{
		{"go", "go.mod", "go"},
		{"node", "package.json", "node"},
		{"rust", "Cargo.toml", "rust"},
		{"python-pyproject", "pyproject.toml", "python"},
		{"python-requirements", "requirements.txt", "python"},
		{"python-pipfile", "Pipfile", "python"},
		{"ruby", "Gemfile", "ruby"},
		{"java-maven", "pom.xml", "java-maven"},
		{"java-gradle", "build.gradle.kts", "java-gradle"},
		{"docker-compose", "compose.yaml", "docker-compose"},
		// Additional language / build-tool ecosystems.
		{"swift-spm", "Package.swift", "swift"},
		{"php", "composer.json", "php"},
		{"scala-sbt", "build.sbt", "scala-sbt"},
		{"scala-mill", "build.mill", "scala-mill"},
		{"scala-mill-sc", "build.sc", "scala-mill"},
		{"cmake", "CMakeLists.txt", "cmake"},
		{"autotools-configure-ac", "configure.ac", "autotools"},
		{"autotools-makefile-am", "Makefile.am", "autotools"},
		{"r-description", "DESCRIPTION", "r"},
		{"zig", "build.zig", "zig"},
		{"zig-zon", "build.zig.zon", "zig"},
		{"perl-makefile-pl", "Makefile.PL", "perl"},
		{"perl-cpanfile", "cpanfile", "perl"},
		{"perl-dist-ini", "dist.ini", "perl"},
		// Static-site generators.
		{"hugo-toml", "hugo.toml", "hugo"},
		{"hugo-yaml", "hugo.yaml", "hugo"},
		{"jekyll", "_config.yml", "jekyll"},
		{"eleventy-dotjs", ".eleventy.js", "eleventy"},
		{"eleventy-config", "eleventy.config.mjs", "eleventy"},
		{"astro-mjs", "astro.config.mjs", "astro"},
		{"astro-ts", "astro.config.ts", "astro"},
		{"gatsby", "gatsby-config.js", "gatsby"},
		{"mkdocs", "mkdocs.yml", "mkdocs"},
		{"docusaurus", "docusaurus.config.js", "docusaurus"},
		{"pelican", "pelicanconf.py", "pelican"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, tc.indicator), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			matches := projectdetect.Detect(nil, dir)
			if len(matches) != 1 || matches[0].Type != tc.wantType {
				t.Fatalf("Detect: got %+v, want single match for %q", matches, tc.wantType)
			}
			if matches[0].Indicator != tc.indicator {
				t.Errorf("indicator = %q, want %q", matches[0].Indicator, tc.indicator)
			}
		})
	}
}

func TestDetect_GlobIndicators(t *testing.T) {
	cases := []struct {
		name     string
		file     string
		wantType string
	}{
		{"terraform", "main.tf", "terraform"},
		{"terraform-variants", "providers.tf", "terraform"},
		{"dotnet-csproj", "MyApp.csproj", "dotnet"},
		{"dotnet-fsproj", "MyApp.fsproj", "dotnet"},
		{"dotnet-sln", "MyApp.sln", "dotnet"},
		{"dotnet-slnx", "MyApp.slnx", "dotnet"},
		{"dotnet-slnf", "MyApp.slnf", "dotnet"},
		{"dotnet-global-json", "global.json", "dotnet"},
		{"dotnet-directory-build-props", "Directory.Build.props", "dotnet"},
		{"dotnet-directory-packages-props", "Directory.Packages.props", "dotnet"},
		{"dotnet-nuget-config", "nuget.config", "dotnet"},
		// HasFile matching is case-insensitive (equalFold) — the
		// conventional NuGet capitalisation must still detect.
		{"dotnet-nuget-config-mixed-case", "NuGet.Config", "dotnet"},
		{"swift-podspec", "Alamofire.podspec", "swift"},
		{"r-rproj", "myanalysis.Rproj", "r"},
		{"matlab-prj", "Export_fig.prj", "matlab"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, tc.file), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			matches := projectdetect.Detect(nil, dir)
			if len(matches) != 1 || matches[0].Type != tc.wantType {
				t.Fatalf("Detect: got %+v, want single match for %q", matches, tc.wantType)
			}
		})
	}
}

// TestDetect_XcodeSubdirOnly covers issue #3: a pure Xcode repo whose
// only marker is a *.xcodeproj / *.xcworkspace bundle (a DIRECTORY) and
// that ships no Package.swift. HasFile / HasGlob only see files, so the
// HasSubdirGlob indicator is what makes this detect as swift.
func TestDetect_XcodeSubdirOnly(t *testing.T) {
	cases := []struct {
		name      string
		bundleDir string
		indicator string
	}{
		{"xcodeproj", "MyApp.xcodeproj", "*.xcodeproj/"},
		{"xcworkspace", "MyApp.xcworkspace", "*.xcworkspace/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			// A directory bundle, plus an unrelated file to prove the
			// file-based indicators do NOT fire on it.
			mustMkdir(t, filepath.Join(dir, tc.bundleDir))
			mustWrite(t, filepath.Join(dir, "README.md"), "x")
			matches := projectdetect.Detect(nil, dir)
			if len(matches) != 1 || matches[0].Type != "swift" {
				t.Fatalf("Detect: got %+v, want single swift match", matches)
			}
			if matches[0].Indicator != tc.indicator {
				t.Errorf("indicator = %q, want %q", matches[0].Indicator, tc.indicator)
			}
		})
	}
}

// TestDetect_SubdirGlobIgnoresFiles guards the files/subdirs split: a
// HasSubdirGlob must NOT match a same-named regular FILE. A stray file
// literally called "x.xcodeproj" should not make a directory detect as
// swift.
func TestDetect_SubdirGlobIgnoresFiles(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "x.xcodeproj"), "not a real bundle")
	if matches := projectdetect.Detect(nil, dir); len(matches) != 0 {
		t.Errorf("Detect: got %+v, want empty (a *.xcodeproj FILE is not a subdir marker)", matches)
	}
}

func TestDetect_MultipleTypes(t *testing.T) {
	// A Go module that also ships a docker-compose.yml fires both
	// — matches the cross-firing semantics established in PR #95.
	dir := t.TempDir()
	for _, f := range []string{"go.mod", "docker-compose.yml"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	matches := projectdetect.Detect(nil, dir)
	got := make([]string, len(matches))
	for i, m := range matches {
		got[i] = m.Type
	}
	sort.Strings(got)
	want := []string{"docker-compose", "go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Detect: got %v, want %v", got, want)
	}
}

// TestDetect_DotnetSlnxRoot mirrors a real modern .NET layout (e.g.
// ~/Code/SPAN/Cel2Sql.NET): the repo ROOT carries only an XML .slnx
// solution plus Directory.*.props, while every *.csproj lives in a
// subdirectory. Before .slnx / MSBuild-marker support the root matched
// nothing (no .sln, no root .csproj). It must now detect as dotnet.
func TestDetect_DotnetSlnxRoot(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"Cel2Sql.slnx", "Directory.Build.props", "Directory.Packages.props"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	matches := projectdetect.Detect(nil, dir)
	if len(matches) != 1 || matches[0].Type != "dotnet" {
		t.Fatalf("Detect: got %+v, want single dotnet match for a .slnx-only root", matches)
	}
}

func TestDetect_NoMatch(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "random.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if matches := projectdetect.Detect(nil, dir); len(matches) != 0 {
		t.Errorf("Detect: got %+v, want empty", matches)
	}
}

func TestFind_StopsAtProjectRoot(t *testing.T) {
	root := t.TempDir()
	// root/a/go.mod  → a is a project root
	// root/a/inner/go.mod  → would be a project but Find should stop at /a
	mustMkdir(t, filepath.Join(root, "a"))
	mustMkdir(t, filepath.Join(root, "a", "inner"))
	mustWrite(t, filepath.Join(root, "a", "go.mod"), "module a\n")
	mustWrite(t, filepath.Join(root, "a", "inner", "go.mod"), "module a/inner\n")

	result, err := projectdetect.Find(t.Context(), root, projectdetect.FindOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 {
		t.Fatalf("got %d projects, want 1; projects=%+v", result.Count, result.Projects)
	}
	if got := result.Projects[0].Path; got != filepath.Join(root, "a") {
		t.Errorf("path = %q, want a", got)
	}
}

func TestFind_NestedTrue(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "a"))
	mustMkdir(t, filepath.Join(root, "a", "inner"))
	mustWrite(t, filepath.Join(root, "a", "go.mod"), "module a\n")
	mustWrite(t, filepath.Join(root, "a", "inner", "Cargo.toml"), "[package]\nname=\"x\"\n")

	result, err := projectdetect.Find(t.Context(), root, projectdetect.FindOptions{Nested: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 2 {
		t.Fatalf("got %d projects, want 2; projects=%+v", result.Count, result.Projects)
	}
}

func TestFind_TypesFilter(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "go-app"))
	mustMkdir(t, filepath.Join(root, "rust-app"))
	mustMkdir(t, filepath.Join(root, "node-app"))
	mustWrite(t, filepath.Join(root, "go-app", "go.mod"), "module x\n")
	mustWrite(t, filepath.Join(root, "rust-app", "Cargo.toml"), "[package]\nname=\"x\"\n")
	mustWrite(t, filepath.Join(root, "node-app", "package.json"), `{"name":"x"}`)

	result, err := projectdetect.Find(t.Context(), root, projectdetect.FindOptions{
		Types: []string{"go", "rust"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 2 {
		t.Fatalf("got %d projects, want 2; projects=%+v", result.Count, result.Projects)
	}
}

func TestFind_Excludes(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "real"))
	mustMkdir(t, filepath.Join(root, "node_modules"))
	mustMkdir(t, filepath.Join(root, "node_modules", "vendored"))
	mustWrite(t, filepath.Join(root, "real", "go.mod"), "module real\n")
	mustWrite(t, filepath.Join(root, "node_modules", "vendored", "go.mod"), "module vendored\n")

	result, err := projectdetect.Find(t.Context(), root, projectdetect.FindOptions{
		Excludes: []string{"node_modules"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 {
		t.Fatalf("got %d projects, want 1 (node_modules excluded); projects=%+v", result.Count, result.Projects)
	}
}

func TestFind_SkipDir(t *testing.T) {
	root := t.TempDir()
	// root/keep/go.mod   → reported
	// root/skip/go.mod   → pruned by SkipDir predicate
	// root/skip/inner/go.mod → pruned with its parent's subtree
	mustMkdir(t, filepath.Join(root, "keep"))
	mustMkdir(t, filepath.Join(root, "skip", "inner"))
	mustWrite(t, filepath.Join(root, "keep", "go.mod"), "module keep\n")
	mustWrite(t, filepath.Join(root, "skip", "go.mod"), "module skip\n")
	mustWrite(t, filepath.Join(root, "skip", "inner", "go.mod"), "module inner\n")

	var seen []string
	result, err := projectdetect.Find(t.Context(), root, projectdetect.FindOptions{
		Nested: true,
		SkipDir: func(relPath, name string) bool {
			seen = append(seen, relPath)
			return name == "skip"
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 || result.Projects[0].Path != filepath.Join(root, "keep") {
		t.Fatalf("got %+v, want only the 'keep' project (skip subtree pruned)", result.Projects)
	}
	// The predicate must receive relPath relative to root, and must NOT
	// have been asked about anything below the pruned subtree.
	for _, rel := range seen {
		if rel == filepath.Join("skip", "inner") {
			t.Errorf("SkipDir was consulted for %q inside a pruned subtree", rel)
		}
		if filepath.IsAbs(rel) {
			t.Errorf("SkipDir relPath = %q, want a path relative to root", rel)
		}
	}
}

func TestFind_PrunesVCSDirsByDefault(t *testing.T) {
	root := t.TempDir()
	// A go.mod buried inside .git must not surface by default.
	mustMkdir(t, filepath.Join(root, ".git", "sub"))
	mustWrite(t, filepath.Join(root, ".git", "sub", "go.mod"), "module ghost\n")
	mustMkdir(t, filepath.Join(root, "real"))
	mustWrite(t, filepath.Join(root, "real", "go.mod"), "module real\n")

	result, err := projectdetect.Find(t.Context(), root, projectdetect.FindOptions{Nested: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 || result.Projects[0].Path != filepath.Join(root, "real") {
		t.Fatalf("got %+v, want only 'real' (.git pruned by default)", result.Projects)
	}

	// Opt back in: now the buried project is visible.
	result, err = projectdetect.Find(t.Context(), root, projectdetect.FindOptions{
		Nested:         true,
		IncludeVCSDirs: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 2 {
		t.Fatalf("got %d projects, want 2 with IncludeVCSDirs=true; projects=%+v", result.Count, result.Projects)
	}
}

func TestCollectBuildExcludesWithOptions_SkipDir(t *testing.T) {
	root := t.TempDir()
	// A Go module (BuildExcludes include "vendor") under a subtree the
	// caller prunes — its excludes must not be collected.
	mustMkdir(t, filepath.Join(root, "skip"))
	mustWrite(t, filepath.Join(root, "skip", "go.mod"), "module skip\n")
	// A Node project that is kept (BuildExcludes include "node_modules").
	mustMkdir(t, filepath.Join(root, "keep"))
	mustWrite(t, filepath.Join(root, "keep", "package.json"), `{"name":"x"}`)

	excludes, err := projectdetect.CollectBuildExcludesWithOptions(t.Context(), root, projectdetect.FindOptions{
		SkipDir: func(_, name string) bool { return name == "skip" },
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, ex := range excludes {
		if ex == "vendor" {
			t.Errorf("collected %q from a pruned subtree; excludes=%v", ex, excludes)
		}
	}
	gotNodeModules := false
	for _, ex := range excludes {
		if ex == "node_modules" {
			gotNodeModules = true
		}
	}
	if !gotNodeModules {
		t.Errorf("excludes=%v, want node_modules from the kept project", excludes)
	}
}

func TestRegistry_Types(t *testing.T) {
	types := projectdetect.DefaultRegistry().Types()
	if len(types) < 28 {
		t.Errorf("registered types = %d, want at least 28 built-ins (10 original + 8 SSGs + 10 language ecosystems)", len(types))
	}
	// Verify sorted by Name.
	for i := 1; i < len(types); i++ {
		if types[i-1].Name > types[i].Name {
			t.Errorf("types not sorted: %s > %s", types[i-1].Name, types[i].Name)
		}
	}
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
