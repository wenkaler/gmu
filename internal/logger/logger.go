package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Setup initializes the logger with the desired verbosity.
func Setup(verbose bool) *slog.Logger {
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize Time format
			if a.Key == slog.TimeKey {
				t := a.Value.Time()
				return slog.String(a.Key, t.Format("2006-01-02 15:04:05"))
			}
			// Customize Level to Uppercase
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				return slog.String(a.Key, strings.ToUpper(level.String()))
			}
			return a
		},
	}

	return slog.New(slog.NewJSONHandler(os.Stderr, opts))
}
