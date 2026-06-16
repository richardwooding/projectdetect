// Package celindicators provides the CEL backend for projectdetect's
// CELExpr indicators. Importing it (typically as a blank import) installs
// the compiler so YAML-loaded custom project types can use `cel:`
// indicators:
//
//	import _ "github.com/richardwooding/projectdetect/celindicators"
//
// The base projectdetect package has no CEL dependency; only this
// sub-package pulls in cel-go, so consumers that need only the built-ins
// and HasFile / HasGlob indicators don't link it.
package celindicators

import (
	"fmt"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"

	"github.com/richardwooding/projectdetect"
)

func init() {
	projectdetect.SetCELCompiler(Compile)
}

// dirCELEnv is the cel.Env every project-type CELExpr indicator compiles
// against. Two list<string> variables:
//
//	files    — basenames of files in the inspected directory
//	subdirs  — basenames of immediate subdirectories
//
// Standard CEL operators (in, exists, endsWith, startsWith, matches, size,
// …) cover the vocabulary. Built lazily once, then reused.
var (
	dirCELEnvOnce sync.Once
	dirCELEnv     *cel.Env
	dirCELEnvErr  error
)

func dirEnv() (*cel.Env, error) {
	dirCELEnvOnce.Do(func() {
		dirCELEnv, dirCELEnvErr = cel.NewEnv(
			cel.Variable("files", cel.ListType(cel.StringType)),
			cel.Variable("subdirs", cel.ListType(cel.StringType)),
		)
	})
	return dirCELEnv, dirCELEnvErr
}

// program wraps a compiled cel.Program as a projectdetect.DirEvaluator.
type program struct{ prog cel.Program }

// Eval runs the program against (files, subdirs) and returns true iff it
// resolves to CEL boolean true. Any error (type mismatch, runtime issue)
// is swallowed as false — an indicator that can't evaluate cleanly
// shouldn't fire.
func (p program) Eval(files, subdirs []string) bool {
	out, _, err := p.prog.Eval(map[string]any{
		"files":   files,
		"subdirs": subdirs,
	})
	if err != nil {
		return false
	}
	return out == types.True
}

// Compile parses and type-checks expr against the directory CEL env and
// returns a projectdetect.DirEvaluator. It is the function installed via
// projectdetect.SetCELCompiler.
func Compile(expr string) (projectdetect.DirEvaluator, error) {
	env, err := dirEnv()
	if err != nil {
		return nil, fmt.Errorf("project-type CEL env: %w", err)
	}
	ast, issues := env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("compile %q: %w", expr, issues.Err())
	}
	prog, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("program %q: %w", expr, err)
	}
	return program{prog: prog}, nil
}
