package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

func parseGoMod(content string) ([]Dependency, map[string]bool) {
	var dependencies []Dependency
	replaces := make(map[string]bool)

	lines := strings.Split(content, "\n")
	inRequire := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "require ("):
			inRequire = true
		case strings.HasPrefix(line, "require ") && !inRequire:
			line = strings.TrimPrefix(line, "require ")
			if dep := parseRequireLine(line); dep != nil {
				dependencies = append(dependencies, *dep)
			}
		case inRequire && line == ")":
			inRequire = false
		case inRequire:
			if dep := parseRequireLine(line); dep != nil {
				dependencies = append(dependencies, *dep)
			}
		case strings.HasPrefix(line, "replace "):
			if path := extractReplacePath(line); path != "" {
				replaces[path] = true
			}
		}
	}

	return dependencies, replaces
}

func parseRequireLine(line string) *Dependency {
	if strings.Contains(line, "indirect") {
		return nil
	}

	if idx := strings.Index(line, "//"); idx != -1 {
		line = strings.TrimSpace(line[:idx])
	}

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil
	}

	return &Dependency{
		Path:    parts[0],
		Version: parts[1],
	}
}

func extractReplacePath(line string) string {
	parts := strings.Split(line, "=>")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(parts[0], "replace "))
}

func replaceInGoMod(module, localPath string) error {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return fmt.Errorf("failed to read go.mod: %w", err)
	}

	// Check if the replace directive already exists
	_, replaces := parseGoMod(string(content))
	if _, ok := replaces[module]; ok {
		fmt.Printf("%sReplace directive for %s%s%s already exists.%s\n", ColorYellow, ColorGreen, module, ColorYellow, ColorReset)
		return nil
	}

	newReplace := fmt.Sprintf("\nreplace %s => %s\n", module, localPath)

	tmpFile := "go.mod.tmp"
	if err := os.WriteFile(tmpFile, append(content, newReplace...), 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpFile, "go.mod"); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

func clearReplaces() error {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}

	var buffer bytes.Buffer
	lines := strings.Split(string(content), "\n")
	inReplaceSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inReplaceSection {
			if strings.HasPrefix(trimmed, ")") {
				inReplaceSection = false
			}
			continue
		}

		if strings.HasPrefix(trimmed, "replace (") {
			inReplaceSection = true
			continue
		}

		if strings.HasPrefix(trimmed, "replace") {
			continue
		}

		buffer.WriteString(line)
		buffer.WriteByte('\n')
	}

	
	tmpFile := "go.mod.tmp"
	if err := os.WriteFile(tmpFile, buffer.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	return os.Rename(tmpFile, "go.mod")
}

func printReplacesFunc() error {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	inReplaceSection := false
	var replaces []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inReplaceSection {
			if strings.HasPrefix(trimmed, ")") {
				inReplaceSection = false
				continue
			}
			replaces = append(replaces, trimmed)
			continue
		}

		if strings.HasPrefix(trimmed, "replace (") {
			inReplaceSection = true
			continue
		}

		if strings.HasPrefix(trimmed, "replace") {
			replaces = append(replaces, strings.TrimPrefix(trimmed, "replace "))
		}
	}

	if len(replaces) > 0 {
		fmt.Printf("%sFound replaces:%s\n", ColorYellow, ColorReset)
		for _, r := range replaces {
			fmt.Printf(" %s%s%s\n", ColorGreen, r, ColorReset)
		}
	} else {
		fmt.Println("No replace directives found in go.mod")
	}

	return nil
}
