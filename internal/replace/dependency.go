package replace

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/wenkaler/gmu/internal/logger"
)

func findLocalPath(modulePath string) (string, error) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		// Fallback to default GOPATH
		home, err := os.UserHomeDir()
		if err == nil {
			gopath = filepath.Join(home, "go")
		}
	}

	originalPath := filepath.Join(gopath, "src", modulePath)
	if _, err := os.Stat(originalPath); err == nil {
		return originalPath, nil
	}

	basePath := removeVersionFromPath(modulePath)
	if basePath != modulePath {
		versionlessPath := filepath.Join(gopath, "src", basePath)
		if _, err := os.Stat(versionlessPath); err == nil {
			return versionlessPath, nil
		}
	}

	return "", fmt.Errorf("local copy not found: tried %s and %s", originalPath,
		filepath.Join(gopath, "src", basePath))
}

func removeVersionFromPath(path string) string {
	re := regexp.MustCompile(`(/v\d+)$`)
	return re.ReplaceAllString(path, "")
}

func selectDependency(matched []string) (string, error) {
	if len(matched) == 1 {
		return matched[0], nil
	}

	fmt.Printf("\n%sMultiple matches found:%s\n", logger.ColorYellow, logger.ColorReset)
	for i, m := range matched {
		fmt.Printf("%s%d) %s%s\n", logger.ColorBlue, i+1, m, logger.ColorReset)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%sEnter the number of the desired package:%s ", logger.ColorYellow, logger.ColorReset)
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
	fmt.Printf("\n%sYou selected:%s %s%s%s\n", logger.ColorYellow, logger.ColorReset, logger.ColorGreen, selected, logger.ColorReset)
	fmt.Printf("%sConfirm selection (press Enter to continue, any other key to cancel):%s ", logger.ColorYellow, logger.ColorReset)
	confirm, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(confirm) == ""
}
