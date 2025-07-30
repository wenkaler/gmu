package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Dependency struct {
	Path    string
	Version string
}

func findLocalPath(modulePath string) (string, error) {
	originalPath := filepath.Join(os.Getenv("GOPATH"), "src", modulePath)
	if _, err := os.Stat(originalPath); err == nil {
		return originalPath, nil
	}

	basePath := removeVersionFromPath(modulePath)
	if basePath != modulePath {
		versionlessPath := filepath.Join(os.Getenv("GOPATH"), "src", basePath)
		if _, err := os.Stat(versionlessPath); err == nil {
			return versionlessPath, nil
		}
	}

	return "", fmt.Errorf("local copy not found: tried %s and %s", originalPath,
		filepath.Join(os.Getenv("GOPATH"), "src", basePath))
}

func removeVersionFromPath(path string) string {
	re := regexp.MustCompile(`(/v\d+)$`)
	return re.ReplaceAllString(path, "")
}

func filterDependencies(deps []Dependency, replaces map[string]bool, partialName string) []string {
	var matched []string
	for _, dep := range deps {
		if replaces[dep.Path] {
			continue
		}
		if strings.Contains(dep.Path, partialName) {
			matched = append(matched, dep.Path)
		}
	}
	return matched
}

func selectDependency(matched []string) (string, error) {
	if len(matched) == 1 {
		return matched[0], nil
	}

	fmt.Printf("\n%sMultiple matches found:%s\n", ColorYellow, ColorReset)
	for i, m := range matched {
		fmt.Printf("%s%d) %s%s\n", ColorBlue, i+1, m, ColorReset)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%sEnter the number of the desired package:%s ", ColorYellow, ColorReset)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input")
	}

	input = strings.TrimSpace(input)
	if len(input) > 256 {
		return "", fmt.Errorf("input too long")
	}

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(matched) {
		return "", fmt.Errorf("invalid selection")
	}

	return matched[idx-1], nil
}

func confirmSelection(selected string) bool {
	fmt.Printf("\n%sYou selected:%s %s%s%s\n", ColorYellow, ColorReset, ColorGreen, selected, ColorReset)
	fmt.Printf("%sConfirm selection (press Enter to continue, any other key to cancel):%s ", ColorYellow, ColorReset)
	confirm, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(confirm) == ""
}