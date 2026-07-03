package projectdetect_test

import (
	"strings"
	"testing"

	"github.com/richardwooding/projectdetect"
)

func TestIsVendored(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		// Minified assets / source maps.
		{"docs/js/jquery.min.js", true},
		{"assets/app-min.css", true},
		{"static/app.js.map", true},
		// Dependency / vendor directories at any depth.
		{"node_modules/left-pad/index.js", true},
		{"web/bower_components/x/x.js", true},
		{"vendor/github.com/foo/bar.go", true},
		{"src/Vendor/thing.js", true}, // case-insensitive
		{"third_party/zlib/zlib.c", true},
		{"a/b/externals/lib.js", true},
		// Build output / caches.
		{"build/dist/app.js", true},
		{"tmp/cache/data.js", true},
		// Well-known libraries by name (not necessarily minified).
		{"website/scripts/prism.js", true},
		{"assets/bootstrap.css", true},
		{"js/angular.js", true},
		{"css/normalize.css", true},
		// Genuine source — must NOT match.
		{"src/main.go", false},
		{"internal/content/sourcetype.go", false},
		{"app.js", false},           // bare app.js is real source
		{"lib/parser.py", false},    //
		{"docs/guide.md", false},    // docs themselves aren't vendored
		{"cmd/tool/main.go", false}, //
	}
	for _, tc := range cases {
		if got := projectdetect.IsVendored(tc.path); got != tc.want {
			t.Errorf("IsVendored(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestVendorMatcher_Custom(t *testing.T) {
	// A fresh empty matcher matches nothing until patterns are added.
	m := projectdetect.NewVendorMatcher()
	if m.Match("node_modules/x.js") {
		t.Error("empty matcher should match nothing")
	}
	if err := m.Add(`(^|/)generated/`, `\.pb\.go$`); err != nil {
		t.Fatal(err)
	}
	if !m.Match("api/generated/types.ts") {
		t.Error("custom pattern generated/ should match")
	}
	if !m.Match("proto/user.pb.go") {
		t.Error("custom pattern .pb.go should match")
	}
	if m.Match("src/main.go") {
		t.Error("custom matcher should not match plain source")
	}

	// A bad pattern is reported and nothing is added.
	if err := m.Add(`([`); err == nil {
		t.Error("expected error for invalid regexp")
	}
}

func TestDefaultVendorMatcher_Independent(t *testing.T) {
	// DefaultVendorMatcher returns an independent instance; adding to it must not
	// leak into the package-level IsVendored.
	m := projectdetect.DefaultVendorMatcher()
	if err := m.Add(`(^|/)my-unique-marker/`); err != nil {
		t.Fatal(err)
	}
	if !m.Match("x/my-unique-marker/f.js") {
		t.Error("added pattern should match on the local matcher")
	}
	if projectdetect.IsVendored("x/my-unique-marker/f.js") {
		t.Error("adding to a DefaultVendorMatcher instance must not affect IsVendored")
	}
}

func TestIsMinified(t *testing.T) {
	// A realistic minified blob: one very long line.
	min := []byte("!function(e){" + strings.Repeat("var a=1;b(a);c(a,b);", 200) + "}(window);")
	if !projectdetect.IsMinified(min) {
		t.Errorf("expected minified content to be detected (len=%d)", len(min))
	}

	// Normal source: many short lines.
	var sb strings.Builder
	for i := 0; i < 200; i++ {
		sb.WriteString("func doThing() { return 1 }\n")
	}
	if projectdetect.IsMinified([]byte(sb.String())) {
		t.Error("normal multi-line source should not be flagged minified")
	}

	// Too small to judge.
	if projectdetect.IsMinified([]byte("x=1;")) {
		t.Error("tiny content should never be minified")
	}

	// Binary content (NUL byte, few newlines) must not be flagged minified.
	bin := make([]byte, 2000)
	bin[42] = 0x00
	if projectdetect.IsMinified(bin) {
		t.Error("binary content should not be flagged minified")
	}
}
