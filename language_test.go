package projectdetect_test

import (
	"sort"
	"testing"

	"github.com/richardwooding/projectdetect"
)

func TestLanguageForPath(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		// One representative per language id.
		{"main.go", "go"},
		{"app.py", "python"},
		{"index.js", "javascript"},
		{"index.tsx", "typescript"},
		{"Main.java", "java"},
		{"lib.rs", "rust"},
		{"main.c", "c"},
		{"widget.cpp", "cpp"},
		{"Program.cs", "csharp"},
		{"Main.kt", "kotlin"},
		{"index.php", "php"},
		{"app.rb", "ruby"},
		{"Main.scala", "scala"},
		{"analysis.r", "r"},
		{"solve.m", "matlab"},
		{"script.pl", "perl"},
		{"App.swift", "swift"},

		// Extra extensions for languages with several.
		{"types.pyi", "python"},
		{"fast.pyx", "python"},
		{"module.mjs", "javascript"},
		{"defs.d.ts", "typescript"}, // trailing ".ts" wins
		{"header.hpp", "cpp"},
		{"template.tcc", "cpp"},
		{"build.gradle.kts", "kotlin"},
		{"legacy.php5", "php"},
		{"tasks.rake", "ruby"},
		{"build.sc", "scala"},
		{"Module.pm", "perl"},

		// Ambiguous extensions resolve to the documented single language.
		{"header.h", "c"},

		// Unrecognised or extension-less paths.
		{"README", ""},
		{"notes.txt", ""},
		{"Gemfile", ""},
		{"archive.tar.gz", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := projectdetect.LanguageForPath(tc.path); got != tc.want {
			t.Errorf("LanguageForPath(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestLanguageForPath_CaseInsensitive(t *testing.T) {
	cases := map[string]string{
		"MAIN.GO":    "go",
		"App.Py":     "python",
		"stats.R":    "r",
		"Widget.CPP": "cpp",
	}
	for path, want := range cases {
		if got := projectdetect.LanguageForPath(path); got != want {
			t.Errorf("LanguageForPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestLanguageForExt_Normalisation(t *testing.T) {
	// Dotted, undotted, and mixed-case forms all resolve the same.
	for _, ext := range []string{"go", ".go", ".GO", "Go"} {
		if got := projectdetect.LanguageForExt(ext); got != "go" {
			t.Errorf("LanguageForExt(%q) = %q, want %q", ext, got, "go")
		}
	}
	if got := projectdetect.LanguageForExt(""); got != "" {
		t.Errorf("LanguageForExt(%q) = %q, want empty", "", got)
	}
	if got := projectdetect.LanguageForExt(".unknown"); got != "" {
		t.Errorf("LanguageForExt(%q) = %q, want empty", ".unknown", got)
	}
}

func TestLanguages_Sorted(t *testing.T) {
	langs := projectdetect.Languages()
	if len(langs) == 0 {
		t.Fatal("Languages() returned nothing")
	}
	if !sort.StringsAreSorted(langs) {
		t.Errorf("Languages() is not sorted: %v", langs)
	}
	for _, lang := range langs {
		if len(projectdetect.ExtensionsForLanguage(lang)) == 0 {
			t.Errorf("language %q has no extensions", lang)
		}
	}
}

// TestMultiExtensionRoundTrip proves the many-extensions-per-language mapping is
// wired both directions: every extension a language claims resolves back to that
// same language, and no extension is claimed by two languages (the reverse map
// is 1:1).
func TestMultiExtensionRoundTrip(t *testing.T) {
	owner := map[string]string{} // ext -> first language seen
	for _, lang := range projectdetect.Languages() {
		exts := projectdetect.ExtensionsForLanguage(lang)
		if !sort.StringsAreSorted(exts) {
			t.Errorf("ExtensionsForLanguage(%q) not sorted: %v", lang, exts)
		}
		for _, ext := range exts {
			if got := projectdetect.LanguageForExt(ext); got != lang {
				t.Errorf("round-trip: ExtensionsForLanguage(%q) lists %q but LanguageForExt(%q) = %q", lang, ext, ext, got)
			}
			if prev, dup := owner[ext]; dup {
				t.Errorf("extension %q claimed by both %q and %q", ext, prev, lang)
			}
			owner[ext] = lang
		}
	}
}

func TestExtensionsForLanguage_Unknown(t *testing.T) {
	if got := projectdetect.ExtensionsForLanguage("cobol"); got != nil {
		t.Errorf("ExtensionsForLanguage(unknown) = %v, want nil", got)
	}
}
