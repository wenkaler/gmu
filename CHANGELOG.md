# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- New `-p` flag to print all replace directives from `go.mod`.
- A check to prevent adding duplicate replace directives.

### Changed
- Refactored the codebase by splitting `main.go` into multiple files for better readability and maintainability (`main.go`, `parser.go`, `dependency.go`, `color.go`).
