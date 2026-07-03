package projectdetect

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"sync"
)

// Vendored-file detection is a third detection axis, complementary to the other
// two: project types answer "what kind of project is this directory", languages
// answer "what language is this file", and this answers "is this file
// third-party / generated content a source-analysis tool should skip".
//
// It differs from a project type's BuildExcludes (which prune whole directories
// by basename): a VendorMatcher classifies individual file paths, so it catches
// a bundled jquery.min.js or a vendored library committed anywhere in the tree —
// including under docs/ — where no directory marker would fire.

// VendorMatcher tests a file path against a set of patterns identifying
// vendored, minified, or otherwise third-party content. Matching is purely
// path-based (no file I/O) and runs against the slash-separated path, so it is
// cross-platform. The zero value is not usable; construct one with
// NewVendorMatcher or DefaultVendorMatcher.
type VendorMatcher struct {
	mu       sync.RWMutex
	patterns []*regexp.Regexp
}

// defaultVendorPatterns is a curated, high-value subset of the patterns GitHub
// Linguist uses to flag vendored content (github.com/github-linguist/linguist,
// lib/linguist/vendor.yml) — minified assets, dependency/vendor directories,
// build output, and well-known front-end libraries. Kept deliberately small and
// conservative to avoid skipping genuine source; extend it per call with
// VendorMatcher.Add or globally with RegisterVendorPattern.
var defaultVendorPatterns = []string{
	// Minified assets and source maps.
	`(\.|-)min\.(js|css)$`,
	`\.(js|css)\.map$`,

	// Dependency / vendor directories, at any depth.
	`(^|/)node_modules/`,
	`(^|/)bower_components/`,
	`(^|/)jspm_packages/`,
	`(?i)(^|/)vendors?/`,
	`(?i)(^|/)(3rd|third)[-_]?party/`,
	`(?i)(^|/)extern(als?)?/`,
	`(^|/)Godeps/`,

	// Build output / caches.
	`(^|/)dist/`,
	`(^|/)cache/`,

	// Well-known vendored front-end libraries, by filename.
	`(?i)(^|/)jquery([.-][^/]*)?\.(js|css)$`,
	`(?i)(^|/)jquery[.-]ui([.-][^/]*)?\.(js|css)$`,
	`(?i)(^|/)prism([.-][^/]*)?\.js$`,
	`(?i)(^|/)bootstrap([.-][^/]*)?\.(js|css|less|scss)$`,
	`(?i)(^|/)modernizr([.-][^/]*)?\.js$`,
	`(?i)(^|/)(angular|react|vue|d3|lodash|underscore|moment|backbone|ember|three|mathjax|highlight|hljs|ace|codemirror|popper|slick|select2|chosen|datatables)([.-][^/]*)?\.js$`,
	`(?i)(^|/)(normalize|foundation|skeleton|materialize|bulma|font-?awesome)\.(css|less|scss)$`,
}

// NewVendorMatcher returns an empty matcher — it matches nothing until you Add
// patterns. Use it to opt out of the built-in set entirely.
func NewVendorMatcher() *VendorMatcher {
	return &VendorMatcher{}
}

// precompiledDefaultPatterns compiles the built-in patterns once at package
// init (regexp.Compile is relatively expensive), so DefaultVendorMatcher only
// copies the slice rather than recompiling on every call. A malformed built-in
// panics here at startup — it's a programming bug, not a runtime condition.
var precompiledDefaultPatterns = func() []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, len(defaultVendorPatterns))
	for i, p := range defaultVendorPatterns {
		compiled[i] = regexp.MustCompile(p)
	}
	return compiled
}()

// DefaultVendorMatcher returns a new matcher preloaded with the curated built-in
// patterns (see defaultVendorPatterns). The returned matcher is independent, so
// Add-ing to it does not affect the package-level IsVendored.
func DefaultVendorMatcher() *VendorMatcher {
	m := NewVendorMatcher()
	m.patterns = make([]*regexp.Regexp, len(precompiledDefaultPatterns))
	copy(m.patterns, precompiledDefaultPatterns)
	return m
}

// Add compiles and appends regexp patterns, each matched against the
// slash-separated path. It returns the first compile error and adds nothing in
// that case.
func (m *VendorMatcher) Add(patterns ...string) error {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return fmt.Errorf("projectdetect: invalid vendor pattern %q: %w", p, err)
		}
		compiled = append(compiled, re)
	}
	m.mu.Lock()
	m.patterns = append(m.patterns, compiled...)
	m.mu.Unlock()
	return nil
}

// Match reports whether path matches any registered pattern. Paths are
// normalised to forward slashes before matching.
func (m *VendorMatcher) Match(path string) bool {
	p := filepath.ToSlash(path)
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, re := range m.patterns {
		if re.MatchString(p) {
			return true
		}
	}
	return false
}

// sharedVendor backs the package-level IsVendored / RegisterVendorPattern. It is
// built from the defaults and may be extended process-wide via
// RegisterVendorPattern.
var sharedVendor = DefaultVendorMatcher()

// IsVendored reports whether path looks like vendored/minified/third-party
// content, using the built-in pattern set (plus anything added via
// RegisterVendorPattern). For a matcher you fully control, use a VendorMatcher.
func IsVendored(path string) bool {
	return sharedVendor.Match(path)
}

// RegisterVendorPattern adds patterns to the process-wide matcher used by
// IsVendored. Safe for concurrent use; returns the first compile error.
func RegisterVendorPattern(patterns ...string) error {
	return sharedVendor.Add(patterns...)
}

// minifiedAvgLineLen is the average-bytes-per-line threshold above which a file
// is judged minified. Hand-written source averages well under this (~30–60);
// minified bundles pack thousands of characters onto one or few lines.
const minifiedAvgLineLen = 200

// maxMinifiedScan caps how much of a file IsMinified inspects, so a huge file
// can't turn the check into a performance/DoS sink. 128 KiB is far more than
// enough to judge line shape.
const maxMinifiedScan = 128 * 1024

// IsMinified reports whether content looks minified from its shape alone —
// useful for bundles that carry no telltale name (e.g. a minified prism.js). It
// is a heuristic: content of a meaningful size whose average line length far
// exceeds normal source. Files under 512 bytes are never considered minified,
// and binary content (a NUL byte in the first 512 bytes) is rejected outright
// so images/PDFs/archives — which also have few newlines — aren't mistaken for
// minified text. Only the first maxMinifiedScan bytes are examined.
func IsMinified(content []byte) bool {
	if len(content) < 512 {
		return false
	}
	if len(content) > maxMinifiedScan {
		content = content[:maxMinifiedScan]
	}
	if bytes.IndexByte(content[:512], 0) != -1 {
		return false // looks binary, not minified text
	}
	lines := bytes.Count(content, []byte{'\n'}) + 1
	return len(content)/lines > minifiedAvgLineLen
}
