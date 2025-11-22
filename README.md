# Single-Post Blogging Platform

This project aims to create an incredibly efficient and streamlined single-author blogging platform. The core idea is to serve one and only one blog post at a time, inspired by classic finger/plan files.

## Authoring
Authors simply write their content in Markdown format to a file named `plan.md` in the project root. After saving, the updated content is immediately available on the web.

## Design Philosophy
The platform is designed for optimal performance, minimizing rendering latency to achieve near-instant page loads. The serving stack is built with efficiency in mind, utilizing Go for its speed and concurrency.

## Getting Started

### Prerequisites
- Go 1.20+ installed.

### Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/dewitt/a-simple-plan.git
   cd a-simple-plan
   ```
2. Build the tools:
   ```bash
   go build -o server ./cmd/server
   go build -o plan ./cmd/plan
   ```

### Usage

#### Writing
Edit `plan.md` with your favorite text editor.

#### Previewing
To see how your plan will look:
```bash
./plan preview
```
This will start a local server and open your browser to it.

#### Publishing
To publish your changes (commit and push):
```bash
./plan publish
```

#### Running the Server (Production)
```bash
./server
```
The server listens on port `8080` by default. You can change this by setting the `PORT` environment variable.

## Project Structure
- `DESIGN.md`: Detailed design decisions and architectural overview.
- `TODO.md`: List of outstanding tasks and development progress.
- `README.md`: Project introduction and usage instructions.
- `plan.md`: The single blog plan content (Markdown format).
- `cmd/server/main.go`: The Go HTTP server application.
- `cmd/plan/main.go`: The authoring CLI tool.
- `internal/`: Shared internal packages.