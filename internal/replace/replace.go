package replace

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/wenkaler/gmu/internal/logger"
)

func Run(log *slog.Logger, partialName string) error {
	if len(partialName) > 256 {
		return fmt.Errorf("input too long (max 256 characters)")
	}

	content, err := os.ReadFile("go.mod")
	if err != nil {
		return fmt.Errorf("error reading go.mod: %w", err)
	}

	deps, replaces, err := ParseGoMod(content)
	if err != nil {
		return err
	}

	matched := filterDependencies(deps, replaces, partialName)
	if len(matched) == 0 {
		fmt.Println("No matches found.")
		return nil
	}

	selected, err := selectDependency(matched)
	if err != nil {
		return err
	}

	if !confirmSelection(selected) {
		fmt.Println("Operation canceled.")
		return nil
	}

	localPath, err := findLocalPath(selected)
	if err != nil {
		return err
	}

	if err := AddReplace(selected, localPath); err != nil {
		return fmt.Errorf("failed to update go.mod: %w", err)
	}

	logger.PrintSuccess(fmt.Sprintf("Added replace: %s => %s", selected, localPath))

	// Run go mod tidy
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute go mod tidy: %v\nOutput:\n%s", err, string(output))
	}

	fmt.Printf("go mod tidy command executed successfully:\n%s\n", output)
	return nil
}

func Clear(log *slog.Logger) error {
	if err := ClearReplaces(); err != nil {
		return err
	}
	fmt.Println("All replace directives removed from go.mod")

	// Run go mod tidy
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute go mod tidy: %v\nOutput:\n%s", err, string(output))
	}

	logger.PrintSuccess("Executed: go mod tidy")
	return nil
}

func Show(log *slog.Logger) error {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}

	_, replaces, err := ParseGoMod(content)
	if err != nil {
		return err
	}

	if len(replaces) > 0 {
		fmt.Printf("%sFound replaces:%s\n", logger.ColorYellow, logger.ColorReset)
		for r := range replaces {
			fmt.Printf(" %s%s%s\n", logger.ColorGreen, r, logger.ColorReset)
		}
	} else {
		fmt.Println("No replace directives found in go.mod")
	}

	return nil
}

func filterDependencies(deps []string, replaces map[string]bool, partialName string) []string {
	var matched []string
	for _, dep := range deps {
		if replaces[dep] {
			continue
		}
		if strings.Contains(dep, partialName) {
			matched = append(matched, dep)
		}
	}
	return matched
}
