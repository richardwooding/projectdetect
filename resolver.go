package projectdetect

import (
	"path/filepath"
	"sync"
)

// ProjectResolver answers "what project does this file belong to?"
// by walking up the file's directory chain to the nearest ancestor
// that matches a registered ProjectType. Cached per-directory via
// sync.Map so a walk of 1000 files in one project root costs one
// Detect call, not 1000.
//
// Worker-safe: every concurrent search worker can call Resolve
// without coordination beyond the sync.Map.
type ProjectResolver struct {
	root     string
	registry *Registry
	opts     ResolveOptions
	cache    sync.Map // dir-path → []Match
}

// ResolveOptions tunes which ancestor directories a resolver walk-up
// considers. The zero value matches the historical behaviour except
// that VCS metadata dirs are pruned by default (see IncludeVCSDirs).
type ResolveOptions struct {
	// SkipDir, when non-nil, is consulted for each ancestor directory:
	// returning true skips it as a project-root candidate (the walk
	// continues to its parent). Use it to keep a resolver in step with
	// a caller's own exclusions.
	SkipDir SkipDirFunc
	// IncludeVCSDirs, when false (the default), skips version-control
	// metadata directories (.git, .hg, .svn) as project-root
	// candidates. Set true to consider them.
	IncludeVCSDirs bool
}

// NewResolver returns a resolver rooted at root. Resolve will walk
// up no higher than root — files at the root with no project
// indicators return an empty match slice.
//
// reg may be nil; nil means DefaultRegistry(). VCS metadata dirs are
// pruned by default — use NewResolverWithOptions for a SkipDir
// predicate or to opt back into them.
func NewResolver(root string, reg *Registry) *ProjectResolver {
	return NewResolverWithOptions(root, reg, ResolveOptions{})
}

// NewResolverWithOptions is NewResolver with caller control over which
// ancestor directories are considered during the walk-up.
func NewResolverWithOptions(root string, reg *Registry, opts ResolveOptions) *ProjectResolver {
	if reg == nil {
		reg = defaultRegistry
	}
	return &ProjectResolver{root: filepath.Clean(root), registry: reg, opts: opts}
}

// Resolve walks up from filePath's directory looking for the nearest
// ancestor that matches a registered project type. Returns the
// matched types for the closest project (alphabetically sorted by
// Detect), or nil if no ancestor matches up to the walk root.
//
// Cached: each unique directory is detected at most once for the
// resolver's lifetime.
func (r *ProjectResolver) Resolve(filePath string) []Match {
	dir := filepath.Dir(filePath)
	for {
		if !r.skip(dir) {
			matches := r.detectCached(dir)
			if len(matches) > 0 {
				return matches
			}
		}
		// Stop conditions: reached the walk root, reached filesystem
		// root, or didn't progress (single-segment path on Windows).
		if dir == r.root {
			return nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}

// skip reports whether dir should be passed over as a project-root
// candidate, per the resolver's ResolveOptions. The root is never
// skipped — pruning it would just return no match for any input.
func (r *ProjectResolver) skip(dir string) bool {
	if dir == r.root {
		return false
	}
	base := filepath.Base(dir)
	if !r.opts.IncludeVCSDirs && isVCSDir(base) {
		return true
	}
	if r.opts.SkipDir != nil {
		rel, err := filepath.Rel(r.root, dir)
		if err != nil {
			rel = base
		}
		return r.opts.SkipDir(rel, base)
	}
	return false
}

// detectCached returns the cached match list for dir, populating it
// via registry.Detect on miss. Empty slice cached too — a "definitely
// not a project root" verdict is just as useful for repeat lookups
// as a positive one.
func (r *ProjectResolver) detectCached(dir string) []Match {
	if v, ok := r.cache.Load(dir); ok {
		return v.([]Match)
	}
	matches := r.registry.Detect(nil, dir)
	// LoadOrStore so a concurrent first call doesn't lose its result;
	// the cached slice is immutable so identity doesn't matter beyond
	// "we've decided".
	actual, _ := r.cache.LoadOrStore(dir, matches)
	return actual.([]Match)
}

// ResolveForPath is the single-shot, no-cache, unbounded-walk-up
// counterpart to ProjectResolver.Resolve. Given an absolute file path,
// it walks up the directory chain (no root limit — terminates at the
// filesystem root) and returns the first ancestor that matches a
// registered project type plus the matched types.
//
// Returns ("", nil) when no ancestor matches. Use this for one-off
// "which project does this file belong to?" lookups (e.g. the MCP
// resolve_project_for_path tool, the CLI which-project subcommand).
// For batch walks where multiple files share an ancestor, prefer
// ProjectResolver — it caches per-directory.
//
// reg may be nil; nil means DefaultRegistry(). VCS metadata dirs are
// skipped as candidates by default — use ResolveForPathWithOptions for
// a SkipDir predicate or to opt back into them.
func ResolveForPath(filePath string, reg *Registry) (string, []Match) {
	return ResolveForPathWithOptions(filePath, reg, ResolveOptions{})
}

// ResolveForPathWithOptions is ResolveForPath with caller control over
// which ancestor directories are considered. Because the walk is
// unbounded (no enclosing root), the SkipDir predicate receives the
// directory's full path as relPath.
//
// reg may be nil; nil means DefaultRegistry().
func ResolveForPathWithOptions(filePath string, reg *Registry, opts ResolveOptions) (string, []Match) {
	if reg == nil {
		reg = defaultRegistry
	}
	dir := filepath.Dir(filePath)
	for {
		if !skipResolveDir(dir, opts) {
			matches := reg.Detect(nil, dir)
			if len(matches) > 0 {
				return dir, matches
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

// skipResolveDir reports whether dir should be passed over as a
// project-root candidate during an unbounded ResolveForPath walk-up.
func skipResolveDir(dir string, opts ResolveOptions) bool {
	base := filepath.Base(dir)
	if !opts.IncludeVCSDirs && isVCSDir(base) {
		return true
	}
	if opts.SkipDir != nil {
		// No enclosing root for the unbounded walk: hand the full path.
		return opts.SkipDir(dir, base)
	}
	return false
}
