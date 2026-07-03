package projectdetect

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// Language detection is a separate axis from project-type detection. Project
// types ("go", "node", "java-maven") answer "what kind of project is this
// directory"; a language answers "what language is this single file". The two
// share no machinery — this file is a self-contained extension→language table
// with no dependency on the Registry / Indicator / CEL code.
//
// The canonical language ids match the grammar ids used by the sibling
// tree-sitter tooling (github.com/richardwooding/go-codemetrics/treesitter) so
// a detected id can be passed straight through to a metrics/parse backend
// without translation.

// langExtensions is the source of truth: each language id maps to the file
// extensions (lowercase, leading dot) that identify it. A language commonly has
// several — C++ alone has a dozen header/source spellings — so this is
// deliberately one-to-many. The reverse ext→lang map (extLang) is derived from
// it at init so the two can never drift.
//
// A handful of extensions are ambiguous across languages; each is assigned to
// exactly one id here so the reverse map stays 1:1:
//
//   - .h   → c      (not C++; C is the conservative default for a bare header)
//   - .m   → matlab (not Objective-C; chosen to align with the tree-sitter grammar set)
//   - .sc  → scala  (not a Scala-Mill build script specifically)
//   - .t   → perl   (Perl test files)
//   - .pyx / .pxd → python (Cython)
//
// initExtLang panics if any extension is claimed by two languages, so a future
// edit that introduces a real collision fails loudly at program start rather
// than resolving to whichever entry happened to be iterated last.
var langExtensions = map[string][]string{
	"go":         {".go"},
	"python":     {".py", ".pyw", ".pyi", ".pyx", ".pxd"},
	"javascript": {".js", ".cjs", ".mjs", ".jsx"},
	"typescript": {".ts", ".tsx", ".cts", ".mts"},
	"java":       {".java"},
	"rust":       {".rs"},
	"c":          {".c", ".h"},
	"cpp":        {".cc", ".cpp", ".cxx", ".c++", ".cppm", ".hpp", ".hh", ".hxx", ".h++", ".ipp", ".inl", ".tcc"},
	"csharp":     {".cs", ".csx"},
	"kotlin":     {".kt", ".kts"},
	"php":        {".php", ".phtml", ".php3", ".php4", ".php5", ".phps"},
	"ruby":       {".rb", ".rake", ".gemspec", ".ru"},
	"scala":      {".scala", ".sc"},
	"r":          {".r", ".rprofile"},
	"matlab":     {".m"},
	"perl":       {".pl", ".pm", ".pod", ".t"},
	"swift":      {".swift"},
}

// extLang is the derived reverse index (extension → language id), built once by
// initExtLang. It is read-only after init.
var extLang = initExtLang()

func initExtLang() map[string]string {
	m := make(map[string]string)
	for lang, exts := range langExtensions {
		for _, ext := range exts {
			if prev, dup := m[ext]; dup {
				panic(fmt.Sprintf("projectdetect: extension %q claimed by both %q and %q", ext, prev, lang))
			}
			m[ext] = lang
		}
	}
	return m
}

// LanguageForPath returns the canonical language id for a file path, derived
// from its extension, or "" when the extension is not recognised. Matching is
// case-insensitive, so "Main.GO" and "x.R" resolve like "main.go" and "x.r".
func LanguageForPath(path string) string {
	return LanguageForExt(filepath.Ext(path))
}

// LanguageForExt returns the language id for a single file extension, or "" when
// unrecognised. The argument is normalised, so "go", ".go", and ".GO" are all
// accepted.
func LanguageForExt(ext string) string {
	if ext == "" {
		return ""
	}
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return extLang[ext]
}

// Languages returns the sorted set of language ids known to the detector.
func Languages() []string {
	out := make([]string, 0, len(langExtensions))
	for lang := range langExtensions {
		out = append(out, lang)
	}
	sort.Strings(out)
	return out
}

// ExtensionsForLanguage returns the extensions mapped to a language id, sorted
// and each with a leading dot, or nil for an unknown id. The returned slice is a
// fresh copy the caller may modify.
func ExtensionsForLanguage(lang string) []string {
	exts, ok := langExtensions[lang]
	if !ok {
		return nil
	}
	out := make([]string, len(exts))
	copy(out, exts)
	sort.Strings(out)
	return out
}
