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

	// Remove all replaces
	// Iterating and dropping might be tricky if we modify slice in place,
	// but modfile provides DropReplace.
	// We need to collect what to drop first to avoid index issues if we were iterating indices,
	// but range over slice is safe if we don't modify the slice structure we are ranging over immediately, or?
	// modfile.Replace is a slice pointer.
	// Better: just clear the list?
	// f.Replace = nil ? No, formatting might rely on Cleanup.

	// Let's iterate and call DropReplace for each.
	for _, r := range f.Replace {
		if err := f.DropReplace(r.Old.Path, r.Old.Version); err != nil {
			return fmt.Errorf("dropping replace: %w", err)
		}
	}
	// Do it again to be sure? No, DropReplace should work.
	// Wait, if I iterate `f.Replace` and call `DropReplace`, `DropReplace` modifies `f.Replace`?
	// Yes, `DropReplace` modifies the slice. This is unsafe while iterating.

	// Correct approach: collect params, then drop.
	type replaceParam struct {
		OldPath, OldVersion string
	}
	var toDrop []replaceParam
	for _, r := range f.Replace {
		toDrop = append(toDrop, replaceParam{r.Old.Path, r.Old.Version})
	}

	for _, p := range toDrop {
		if err := f.DropReplace(p.OldPath, p.OldVersion); err != nil {
			return fmt.Errorf("dropping replace %s: %w", p.OldPath, err)
		}
	}

	f.Cleanup() // generic cleanup

	out, err := f.Format()
	if err != nil {
		return fmt.Errorf("formatting go.mod: %w", err)
	}

	return os.WriteFile("go.mod", out, 0644)
}
