# Single-Post Blogging Platform

This project is an incredibly efficient and streamlined single-author blogging platform. It generates a static HTML site from a single Markdown file (`plan.md`), designed for hosting on Cloudflare Pages.

## Authoring
Authors simply write their content in Markdown format to a file named `plan.md` in the project root. After saving, the updated content is immediately available on the web.

## Design Philosophy
The platform is designed for optimal performance, minimizing rendering latency to achieve near-instant page loads. The site is pre-rendered to static HTML and served via a global CDN.

## Getting Started

### Prerequisites
- Go 1.25+ installed.

### Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/dewitt/a-simple-plan.git
   cd a-simple-plan
   ```
2. Build the CLI tool:
   ```bash
   go build -o plan ./cmd/plan
   ```

### Usage

#### Writing
Edit `plan.md` with your favorite text editor.

#### Previewing
To see how your plan will look locally:
```bash
./plan preview
```
This will build the static site, start a local server, and open your browser.

#### Building (Manual)
To generate the static site in the `public/` directory without previewing:
```bash
./plan build
```

#### Publishing
To publish your changes (commit and push to GitHub):
```bash
./plan publish
```
This triggers a deployment on Cloudflare Pages.

## Cloudflare Pages Setup
1.  Connect your GitHub repository to Cloudflare Pages.
2.  Set the **Build command** to: `go run ./cmd/plan build`
3.  Set the **Build output directory** to: `public`
4.  (Optional) Set Environment Variable `GO_VERSION` to `1.25.1`.

## Project Structure
- `DESIGN.md`: Detailed design decisions and architectural overview.
- `TODO.md`: List of outstanding tasks and development progress.
- `README.md`: Project introduction and usage instructions.
- `plan.md`: The single blog plan content (Markdown format).
- `cmd/plan/main.go`: The authoring and building CLI tool.
- `internal/render/`: Shared rendering logic (Markdown -> HTML).