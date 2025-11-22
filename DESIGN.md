# Design Document for the Single-Post Blogging Platform

## 1. Goal
The primary goal is to create an extremely efficient, single-author, single-post blogging platform. It should serve one blog post (a Markdown file) at a time with minimal rendering latency, aiming for near-instant page loads.

## 2. Core Principles
- **Simplicity:** Keep the architecture and codebase as simple as possible.
- **Performance:** Optimize for speed at every layer, especially content delivery and rendering.
- **Ease of Authoring:** Authors should only need to write and save a Markdown file.

## 3. Architecture Proposal

### 3.1. Content Source
- The blog post will be a single Markdown file, conventionally named `plan.md`, located in the project's root directory.

### 3.2. Serving Layer
- A Go-based HTTP server will be used for its performance characteristics and minimal overhead.
- The server will read `plan.md`.
- Upon a request, the server will:
    1. Read the `plan.md` file.
    2. Convert the Markdown content to HTML.
    3. Serve the generated HTML.

### 3.3. Markdown to HTML Conversion
- For initial implementation, a fast, reliable Go Markdown parsing library will be used (e.g., `github.com/gomarkdown/markdown`).
- To further minimize latency, future iterations could consider:
    - Pre-rendering HTML on file save and serving static HTML.
    - Caching rendered HTML in memory.

### 3.4. User Interface (Reader-facing)
- The served HTML will be a minimalist page with basic styling to ensure fast rendering. 
- No client-side JavaScript for content rendering, to avoid parse/execution overhead.
- CSS will be inlined or kept to a minimum via a single `<style>` block to reduce HTTP requests.

### 3.5. Authoring CLI (`plan`)
To streamline the authoring process, a CLI tool named `plan` will be developed.
- **Commands:**
    - `plan preview`: Starts the Go server locally (rendering `plan.md` by default) and automatically opens the user's default web browser to the local URL. This allows the author to see exactly what the reader will see.
    - `plan publish`: Automates the publishing workflow. It will add `plan.md` to the git index, create a commit, and push the changes to the remote repository (GitHub).
    - **Flags:** Both commands should support a flag (e.g., `-f` or `--file`) to specify a different plan file.
- **Shared Logic:** The rendering logic (Markdown -> HTML) will be shared between the main server and the `plan` CLI to ensure the preview matches production.

## 4. Development Workflow
- **Authoring:** The author edits `plan.md`.
- **Previewing:** Run `plan preview` to verify content.
- **Publishing:** Run `plan publish` to deploy.
- **Deployment:** The Go server application will be compiled and run. The production environment (e.g., GitHub Actions or a listening server) will react to the git push.

## 5. Future Considerations (Out of scope for initial MVP)
- **Theming:** More sophisticated styling options.
- **Multiple Posts:** Support for an archive or multiple distinct posts.
- **Admin Interface:** A web-based interface for managing content (though contrary to the "single file" premise).
- **Deployment Automation:** Scripting for continuous deployment.
