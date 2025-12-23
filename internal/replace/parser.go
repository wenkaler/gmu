package replace

import (
	"fmt"
	"os"

	"golang.org/x/mod/modfile"
)

// ParseGoMod returns dependencies (paths) and replaces (path -> true)
func ParseGoMod(content []byte) ([]string, map[string]bool, error) {
	f, err := modfile.Parse("go.mod", content, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing go.mod: %w", err)
	}

	replaces := make(map[string]bool)
	for _, r := range f.Replace {
		replaces[r.Old.Path] = true
	}

	var deps []string
	for _, r := range f.Require {
		deps = append(deps, r.Mod.Path)
	}

	return deps, replaces, nil
}

func AddReplace(module, localPath string) error {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}

	f, err := modfile.Parse("go.mod", content, nil)
	if err != nil {
		return fmt.Errorf("parsing go.mod: %w", err)
	}

	if err := f.AddReplace(module, "", localPath, ""); err != nil {
		return fmt.Errorf("adding replace: %w", err)
	}

	out, err := f.Format()
	if err != nil {
		return fmt.Errorf("formatting go.mod: %w", err)
	}

	return os.WriteFile("go.mod", out, 0644)
}

func ClearReplaces() error {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}

	f, err := modfile.Parse("go.mod", content, nil)
	if err != nil {
		return fmt.Errorf("parsing go.mod: %w", err)
	}

	// Direct syntax manipulation to avoid DropReplace panic and ensure removal.
	// f.Format() uses f.Syntax, not f.Replace struct field.
	var newStmts []modfile.Expr
	for _, stmt := range f.Syntax.Stmt {
		keep := true
		switch x := stmt.(type) {
		case *modfile.Line:
			if len(x.Token) > 0 && x.Token[0] == "replace" {
				keep = false
			}
		case *modfile.LineBlock:
			if len(x.Token) > 0 && x.Token[0] == "replace" {
				keep = false
			}
		}
		if keep {
			newStmts = append(newStmts, stmt)
		}
	}
	f.Syntax.Stmt = newStmts
	f.Replace = nil

	f.Cleanup() // generic cleanup

	out, err := f.Format()
	if err != nil {
		return fmt.Errorf("formatting go.mod: %w", err)
	}

	return os.WriteFile("go.mod", out, 0644)
}
