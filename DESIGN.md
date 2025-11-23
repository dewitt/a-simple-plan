# Design Document for the Single-Post Blogging Platform

## 1. Goal
The primary goal is to create an extremely efficient, single-author, single-post blogging platform. It generates a static HTML site from a Markdown file, enabling zero-latency serving via a CDN (Cloudflare Pages).

## 2. Core Principles
- **Simplicity:** Keep the architecture and codebase as simple as possible.
- **Performance:** Optimize for speed by serving pre-rendered static HTML.
- **Ease of Authoring:** Authors should only need to write and save a Markdown file.

## 3. Architecture Proposal

### 3.1. Content Source
- The blog post will be a single Markdown file, conventionally named `plan.md`, located in the project's root directory.

### 3.2. Serving Layer
- **Static Hosting:** The site is hosted on Cloudflare Pages.
- **No Runtime Server:** There is no dynamic backend. The content is served as a static `index.html` file.

### 3.3. Markdown to HTML Conversion
- A custom CLI tool (`plan`) handles the build process.
- The `plan build` command:
    1. Reads the `plan.md` file.
    2. Converts the Markdown content to HTML using the `goldmark` library.
    3. Injects the HTML into a template.
    4. Writes the result to `public/index.html`.

### 3.4. User Interface (Reader-facing)
- The served HTML is a minimalist page with mid-90s workstation styling (Terminal/Solarized aesthetics).
- CSS is inlined to avoid extra HTTP requests.
- No client-side JavaScript is required for rendering.

### 3.5. Authoring CLI (`plan`)
To streamline the authoring process, a CLI tool named `plan` is used.
- **Commands:**
    - `plan preview`: Builds the static site locally and serves it via a temporary local web server, opening the browser automatically.
    - `plan build`: Generates the static site in the `public/` directory.
    - `plan publish`: Automates the publishing workflow. It adds changes to git, commits, and pushes to GitHub. Cloudflare Pages detects the push and deploys the site.
    - **Flags:** Commands support a `-f` or `--file` flag to specify a different plan file.

## 4. Development Workflow
- **Authoring:** The author edits `plan.md`.
- **Previewing:** Run `plan preview` to verify content locally.
- **Publishing:** Run `plan publish` to push changes to the repository.
- **Deployment:** Cloudflare Pages automatically builds (using `plan build`) and deploys the new version.

## 5. Future Considerations
- **Theming:** More sophisticated styling options.
- **Multiple Posts:** Support for an archive or multiple distinct posts.
- **Asset Management:** Handling images or other static assets alongside the markdown.
