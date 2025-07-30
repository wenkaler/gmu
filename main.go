package main

import (
	"flag"
	"fmt"
	"os"
)

// Version of the application
const Version = "1.2.0"

var (
	showHelp         = flag.Bool("h", false, "Show help")
	showHelpLong     = flag.Bool("help", false, "Show help")
	showVersion      = flag.Bool("version", false, "Show version")
	showShortVersion = flag.Bool("v", false, "Show version")
	clearFlag        = flag.Bool("clear", false, "Remove all replace directives")
	clearFlagShort   = flag.Bool("cl", false, "Short for -clear")
	printReplaces    = flag.Bool("p", false, "Print all replace directives")
	maxInputLen      = 256 // Maximum allowed input length
)

func init() {
	flag.Usage = func() {
		fmt.Printf("%sUsage: goreplace <partial-package-name>%s\n", ColorBlue, ColorReset)
		fmt.Println("Searches for matching dependencies in go.mod and replaces them with local path if found.")
		fmt.Printf("\n%sOptions:%s\n", ColorYellow, ColorReset)
		fmt.Println("  -h, --help      Show this help message")
		fmt.Println("  -v, --version   Show version information")
		fmt.Println("  -cl, --clear    Remove all replace directives")
		fmt.Println("  -p              Print all replace directives")
		fmt.Printf("\n%sExample:%s\n", ColorYellow, ColorReset)
		fmt.Println("  goreplace proto")
	}
}

func main() {
	flag.Parse()

	if *showHelp || *showHelpLong {
		flag.Usage()
		return
	}

	if *showVersion || *showShortVersion {
		fmt.Printf("goreplace version %s%s%s\n", ColorGreen, Version, ColorReset)
		return
	}

	if *printReplaces {
		if err := printReplacesFunc(); err != nil {
			fmt.Printf("Error printing replaces: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Режим очистки replace-директив
	if *clearFlag || *clearFlagShort {
		if err := clearReplaces(); err != nil {
			fmt.Printf("Error clearing replaces: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("All replace directives removed from go.mod")
		printSuccess("You need run: go mod tidy - to update go.sum")
		return
	}

	args := flag.Args()
	if len(args) < 1 {
		printError("missing required argument <partial-package-name>")
		flag.Usage()
		os.Exit(1)
	}

	partialName := args[0]
	if len(partialName) > maxInputLen {
		printError(fmt.Sprintf("input too long (max %d characters)", maxInputLen))
		os.Exit(1)
	}

	modContent, err := os.ReadFile("go.mod")
	if err != nil {
		printError(fmt.Sprintf("error reading go.mod: %v", err))
		os.Exit(1)
	}

	dependencies, replaces := parseGoMod(string(modContent))
	matched := filterDependencies(dependencies, replaces, partialName)

	if len(matched) == 0 {
		fmt.Println("No matches found.")
		return
	}

	selected, err := selectDependency(matched)
	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	if !confirmSelection(selected) {
		fmt.Println("Operation canceled.")
		return
	}

	localPath, err := findLocalPath(selected)
	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	if err := replaceInGoMod(selected, localPath); err != nil {
		printError(fmt.Sprintf("failed to update go.mod: %v", err))
		os.Exit(1)
	}

	printSuccess(fmt.Sprintf("Added replace: %s => %s", selected, localPath))
	printSuccess("You need run: go mod tidy - to update go.sum")
}
