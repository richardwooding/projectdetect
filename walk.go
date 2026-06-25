package projectdetect

import (
	"io/fs"
	"os"
	"path/filepath"
)

// readOSDir is fs.ReadDir's OS-backed counterpart. Kept separate so
// the in-memory test path (fs.FS injection) and the production path
// (direct os.ReadDir) don't share a wrapper.
func readOSDir(dir string) ([]fs.DirEntry, error) {
	return os.ReadDir(dir)
}

// SkipDirFunc reports whether a directory should be pruned from a walk
// (the directory and its whole subtree are skipped). It is consulted
// for every directory except the walk root, which is never pruned.
//
// relPath is the directory's path relative to the walk root (e.g.
// "vendor", "a/node_modules"); name is its base name. For the
// unbounded ResolveForPathWithOptions walk-up there is no enclosing
// root, so relPath is the directory's full path.
//
// Wire it into FindOptions.SkipDir or ResolveOptions.SkipDir to mirror
// an external walker's exclusions — so projectdetect visits exactly the
// same tree as the caller (e.g. file-search-on's own walkers).
type SkipDirFunc func(relPath, name string) bool

// vcsDirs are version-control metadata directories pruned by default
// (when FindOptions.IncludeVCSDirs / ResolveOptions.IncludeVCSDirs is
// false). They never hold project roots, so descending them is wasted
// I/O — and pruning them keeps projectdetect in step with callers that
// already skip VCS metadata in their own walks.
var vcsDirs = map[string]struct{}{
	".git": {},
	".hg":  {},
	".svn": {},
}

// isVCSDir reports whether name is a version-control metadata directory.
func isVCSDir(name string) bool {
	_, ok := vcsDirs[name]
	return ok
}

// excluder is a basename-glob matcher used by Find. Mirrors the
// minimal subset of internal/search/exclude.go's behaviour — full
// gitignore semantics are out of scope for project-root walking,
// where the typical excludes (.git, node_modules, target, dist) are
// just basenames.
type excluder struct {
	patterns []string
}

func newExcluder(patterns []string) *excluder {
	return &excluder{patterns: patterns}
}

func (e *excluder) skip(name string) bool {
	if e == nil {
		return false
	}
	for _, pat := range e.patterns {
		if pat == name {
			return true
		}
		if ok, err := filepath.Match(pat, name); err == nil && ok {
			return true
		}
	}
	return false
}
