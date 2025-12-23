package update

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/wenkaler/gmu/internal/config"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

// UpdateTask represents a single module update operation.
type UpdateTask struct {
	Module  string
	Version string
}

// GoListModule represents the JSON output of 'go list -m -json'.
type GoListModule struct {
	Path    string        `json:"Path"`
	Version string        `json:"Version"`
	Update  *GoListModule `json:"Update"`
}

func Run(logger *slog.Logger, feature string, regexFlag string, excludes []string, doUpdate bool) {
	cfg := config.Load(logger)
	targetPattern := determineRegex(logger, regexFlag, cfg.TargetRegex)
	allExcludes := append(cfg.ExcludePatterns, excludes...)

	logger.Debug("Configuration loaded",
		"feature_suffix", feature,
		"module_regex", targetPattern,
		"excludes", allExcludes,
		"update_mode", doUpdate,
	)

	modFile, err := parseGoMod("go.mod")
	if err != nil {
		logger.Error("Failed to parse go.mod", "error", err)
		return
	}

	reg, err := regexp.Compile(targetPattern)
	if err != nil {
		logger.Error("Invalid regex", "pattern", targetPattern, "error", err)
		return
	}

	var excludeRegs []*regexp.Regexp
	for _, ex := range allExcludes {
		r, err := regexp.Compile(ex)
		if err != nil {
			logger.Warn("Invalid exclude regex, ignoring", "pattern", ex, "error", err)
			continue
		}
		excludeRegs = append(excludeRegs, r)
	}

	logger.Info("Scanning modules...")
	tasks := scanModules(logger, modFile, reg, excludeRegs, feature)

	if len(tasks) == 0 {
		logger.Info("No updates found")
		return
	}

	printUpdates(logger, tasks)

	if doUpdate {
		applyUpdates(logger, tasks)
		runTidy(logger)
	} else {
		logger.Info("Run with -u to apply these changes")
	}
}

func determineRegex(logger *slog.Logger, flagRegex, configRegex string) string {
	if flagRegex != "" {
		return flagRegex
	}
	if configRegex != "" {
		return configRegex
	}
	logger.Warn("No regex provided. Scanning all modules might be slow.")
	return ".*"
}

func parseGoMod(path string) (*modfile.File, error) {
	// Re-reading file content here, or we could pass content.
	// Since we are just renaming functions, let's keep it simple for now but ideally we use x/mod properly.
	// Actually original code used `os.ReadFile` then `modfile.Parse`.
	// Wait, internal/replace/parser.go also parses go.mod but manually.
	// I should probably make a shared `internal/mod` package later if they share logic.
	// For now, I'll keep it here.
	content, err := os.ReadFile(path) // using os directly as in original
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return modfile.Parse(path, content, nil)
}

func scanModules(logger *slog.Logger, f *modfile.File, reg *regexp.Regexp, excludes []*regexp.Regexp, feature string) []UpdateTask {
	var tasks []UpdateTask
	for _, require := range f.Require {
		if !reg.MatchString(require.Mod.Path) {
			continue
		}

		excluded := false
		for _, exReg := range excludes {
			if exReg.MatchString(require.Mod.Path) {
				logger.Debug("Module excluded", "module", require.Mod.Path, "pattern", exReg.String())
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		logger.Debug("Checking module", "module", require.Mod.Path, "current_version", require.Mod.Version)

		var bestVersion string
		var found bool

		if feature != "" {
			bestVersion, found = findFeatureVersion(logger, require.Mod.Path, feature)
		} else {
			bestVersion, found = findLatestStableVersion(logger, require.Mod.Path)
		}

		if !found {
			continue
		}

		if bestVersion != require.Mod.Version {
			tasks = append(tasks, UpdateTask{Module: require.Mod.Path, Version: bestVersion})
		} else {
			logger.Debug("Already on version", "module", require.Mod.Path, "version", bestVersion)
		}
	}
	return tasks
}

func findFeatureVersion(logger *slog.Logger, modulePath, featureSuffix string) (string, bool) {
	cmd := exec.Command("go", "list", "-m", "-versions", modulePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		logger.Warn("Error getting versions", "module", modulePath, "error", err)
		return "", false
	}

	versions := strings.Fields(out.String())
	targetSuffix := fmt.Sprintf("-%s", featureSuffix)
	var bestVersion string

	for _, v := range versions {
		if strings.Contains(v, targetSuffix) {
			if bestVersion == "" || semver.Compare(v, bestVersion) > 0 {
				bestVersion = v
			}
		}
	}

	if bestVersion != "" {
		logger.Debug("Found matching feature version", "module", modulePath, "version", bestVersion)
		return bestVersion, true
	}

	logger.Debug("No match found", "module", modulePath)
	return "", false
}

func findLatestStableVersion(logger *slog.Logger, modulePath string) (string, bool) {
	cmd := exec.Command("go", "list", "-m", "-u", "-json", modulePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		logger.Warn("Error checking for updates", "module", modulePath, "error", err)
		return "", false
	}

	var modInfo GoListModule
	if err := json.Unmarshal(out.Bytes(), &modInfo); err != nil {
		logger.Warn("Failed to parse go list output", "module", modulePath, "error", err)
		return "", false
	}

	if modInfo.Update != nil {
		candidate := modInfo.Update.Version
		if semver.Prerelease(candidate) == "" {
			logger.Debug("Found stable update", "module", modulePath, "version", candidate)
			return candidate, true
		}
		logger.Debug("Update available but not stable", "module", modulePath, "candidate", candidate)
	}

	return "", false
}

func printUpdates(logger *slog.Logger, tasks []UpdateTask) {
	logger.Info("Updates found", "count", len(tasks))
	for _, task := range tasks {
		logger.Info("Update candidate", "module", task.Module, "version", task.Version)
	}
}

func applyUpdates(logger *slog.Logger, tasks []UpdateTask) {
	logger.Info("Applying updates...")
	for _, task := range tasks {
		logger.Info("Updating module", "module", task.Module, "version", task.Version)
		execLog(logger, "go", "get", fmt.Sprintf("%s@%s", task.Module, task.Version))
	}
}

func runTidy(logger *slog.Logger) {
	logger.Info("Running 'go mod tidy'...")
	execLog(logger, "go", "mod", "tidy")
	logger.Info("Tidy complete")
}

func execLog(logger *slog.Logger, command string, args ...string) {
	cmd := exec.Command(command, args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("Failed to create stdout pipe", "error", err)
		return
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		logger.Error("Failed to create stderr pipe", "error", err)
		return
	}

	if err := cmd.Start(); err != nil {
		logger.Error("Failed to start command", "command", command, "error", err)
		return
	}

	scannerOut := bufio.NewScanner(stdoutPipe)
	scannerErr := bufio.NewScanner(stderrPipe)

	go func() {
		for scannerOut.Scan() {
			logger.Info("Command output", "cmd", command, "out", scannerOut.Text())
		}
	}()

	go func() {
		for scannerErr.Scan() {
			logger.Info("Command output", "cmd", command, "out", scannerErr.Text())
		}
	}()

	if err := cmd.Wait(); err != nil {
		logger.Error("Command execution failed", "command", command, "error", err)
	}
}
