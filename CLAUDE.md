# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Glance is a lightweight, self-hosted dashboard application written in Go that displays various feeds (RSS, Reddit, Hacker News, weather, markets, etc.) in a customizable interface. It's a single binary (<20MB) with minimal dependencies, server-side rendering via Go templates, and vanilla JavaScript.

## Build and Run Commands

**Build binary:**
```bash
go build -o build/glance .
```

**Run directly without building:**
```bash
go run .
```

**Build Docker image:**
```bash
docker build -t glance:latest .
```

**Build for specific OS/architecture:**
```bash
GOOS=linux GOARCH=amd64 go build -o build/glance .
```

**Python Environment (uv):**
- `uv sync` - Install/sync dependencies
- `uv add <package>` - Add a new dependency (do not use `pip install`)
- `uv run <command>` - Run a command or script (do not use `source .venv/bin/activate` or `.venv/bin/python`)

**CLI Commands:**
- `glance` - Start server (looks for `glance.yml` in current directory)
- `glance --config <path>` - Use custom config path
- `glance config:validate` - Validate configuration file
- `glance config:print` - Print parsed config with includes expanded
- `glance password:hash <pwd>` - Hash a password for auth config
- `glance secret:make` - Generate random secret key
- `glance sensors:print` - List all available sensors
- `glance diagnose` - Run diagnostic checks

**Testing:**
```bash
go test ./...
```

Note: Test coverage is minimal. Only `auth_test.go` exists currently.

## Architecture

### Entry Point
`main.go` → `internal/glance.Main()` - parses CLI flags and starts the HTTP server.

### Core Application (`internal/glance/glance.go`)
The `application` struct is the central component holding:
- Configuration and pages
- Widget registry and cache
- Authentication state
- HTTP server and mux router
- Theme system

### Widget System

All widgets implement the `widget` interface defined in `internal/glance/widget.go`. Widget implementations are in `widget-*.go` files (28+ widget types).

**Key widget patterns:**
- Each widget has its own update logic, template, and configuration
- Widgets cache data with configurable duration (via `cache` property)
- Outdated widgets update concurrently via goroutines
- Use `widget-utils.go` for shared helpers (HTTP requests with headers, user agents, timeouts)

**Adding a new widget:**
1. Create config struct in `config-fields.go`
2. Create widget struct in a new `widget-<name>.go` file
3. Implement the `widget` interface (update, render logic)
4. Add template in `templates/widget-<name>.html`
5. Register in config parser

### Configuration System (`config.go`)

YAML-based with advanced features:
- Environment variable interpolation: `${VAR}`, `${secret:NAME}`, `${readFileFromEnv:VAR}`
- File includes: `!include: path/to/file.yml`
- Hot-reload via fsnotify (config changes don't require restart)
- Validation on startup and reload

Config files live in `config/` directory (gitignored). See `docs/glance.yml` for example.

### Template System (`templates.go`, `templates/`)

Server-side rendering using Go `html/template`:
- Base template: `document.html`
- Page layout: `page.html`
- Widget templates: `widget-*.html`
- Theme styling via template functions

### Theme System (`theme.go`)

HSL-based color system with CSS custom properties:
- `theme.go` defines color struct and template functions
- Themes configured in YAML with HSL values
- Presets available for light/dark modes
- Custom CSS supported via `user.css` in assets

### Authentication (`auth.go`)

Session-based authentication:
- Bcrypt password hashing
- Token-based sessions with expiration
- Username hashing for privacy
- Rate limiting on failed attempts

### Static Assets (`embed.go`, `static/`)

All static assets embedded in binary:
- CSS bundled at build time (no runtime CSS processing)
- Vanilla JavaScript for interactive features (calendar, masonry, animations)
- Custom fonts and icons
- Cache headers for performance

## Key Patterns and Conventions

1. **No package.json** - Pure Go project, no npm/build scripts
2. **Avoid new dependencies** - Contributing guideline: keep dependency count low
3. **Backward compatibility** - Avoid breaking config changes
4. **Color usage** - Use semantic `primary`, `positive`, `negative` colors, not hard-coded values
5. **Icons** - Use [heroicons.com](https://heroicons.com/) where applicable
6. **Error handling** - Widgets should gracefully handle fetch failures (show errors in UI)
7. **Concurrency** - Widget updates are parallel; use sync primitives carefully

## Development Notes

- Go 1.23+ required
- No CI/CD tests - GitHub Actions only handles releases via GoReleaser
- Multi-architecture support: amd64, arm64, arm/v7, 386
- Config validation happens before server starts
- Widget cache durations vary by type (overridable per widget)
- No periodic background requests - data only fetched on page load

## Contributing Guidelines (from README)

- Base branch: `dev` for new features/bugs, `main` for maintenance
- Submit feature requests before implementing
- Avoid PRs for roadmap/backlog/icebox features
- No `package.json` (still relevant for this Go project)
- Provide screenshots for UI changes
