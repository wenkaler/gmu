package logger

import "fmt"

// Colors for terminal output
const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorReset  = "\033[0m"
)

func PrintError(msg string) {
	fmt.Printf("%sError: %s%s\n", ColorRed, msg, ColorReset)
}

func PrintSuccess(msg string) {
	fmt.Printf("%s%s%s\n", ColorGreen, msg, ColorReset)
}
