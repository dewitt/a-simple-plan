# A Simple Plan

**A Simple Plan** is a minimalist static site generator designed for publishing a personal "plan" or daily log. It turns a single Markdown file (`plan.md`) into a static website with a full history of changes preserved as individual posts.

## Core Philosophy

1.  **Single Source of Truth**: You only ever edit one file: `plan.md`.
2.  **Git-Powered History**: The history of your blog is just the git history of that file.
3.  **Zero Friction**: No front-matter, no file management, no complex configuration.

## Features

*   **Instant Preview**: Run a local server to see your changes as you type.
*   **Automatic Archiving**: The builder uses `git log` to reconstruct the state of your plan for every day it was modified, generating a browsable calendar of your past posts.
*   **Customizable**: Supports a simple `settings.json` and custom HTML templates.
*   **Separation of Concerns**: The builder logic is separate from your content data.

## Installation

```bash
# Clone the repository
git clone https://github.com/dewitt/a-simple-plan.git
cd a-simple-plan

# Install the binary
go install ./cmd/plan
```
*Ensure your `$GOPATH/bin` (usually `~/go/bin`) is in your system `$PATH`.*

## Usage

### 1. Setup Your Plan Repository

Create a new directory (or git repository) for your plan.

```bash
mkdir my-plan
cd my-plan
git init
touch plan.md
```

### 2. Run Commands

All commands can be run from within your plan directory, or by using the `-f` flag to point to it.

```bash
# Preview your site locally at http://localhost:8081
plan preview

# Edit your plan.md in your default editor
plan edit

# Build the static site to the public/ directory
plan build

# Commit changes to git
plan save

# Commit and push changes to origin
plan publish
```

### 3. Configuration (Optional)

Create a `settings.json` file in your plan directory to customize the output.

```json
{
  "username": "dewitt",
  "name": "DeWitt Clinton",
  "directory": "/home/dewitt",
  "shell": "/bin/zsh",
  "timezone": "America/Los_Angeles",
  "title": "My Plan"
}
```

### 4. Templating (Optional)

Create a `template.html` in your plan directory to override the default design.

**Available Placeholders:**
*   `{{content}}`: The rendered Markdown body.
*   `{{onSince}}`: The "created at" date string.
*   `{{modTimeUnix}}`: The modification timestamp (for JavaScript).
*   `{{username}}`, `{{fullname}}`, `{{directory}}`, `{{shell}}`, `{{title}}`: Values from your settings.

## Deployment with Cloudflare Pages

To deploy your `my-plan-repo` content using Cloudflare Pages, follow these steps:

1.  **Make `a-simple-plan` (this repository) Public**: Cloudflare Pages needs to access the builder to run the build command. Go to your GitHub repository settings for `a-simple-plan` and ensure its visibility is set to `Public`.

2.  **Connect `my-plan-repo` to Cloudflare Pages**: In your Cloudflare Dashboard, go to "Pages" and connect your `my-plan-repo` GitHub repository as a new project.

3.  **Configure Build Settings**: In the Cloudflare Pages project settings for `my-plan-repo` (under "Settings" -> "Build & deployments"), update the following:
    *   **Build command**: 
        ```bash
        git clone https://github.com/dewitt/a-simple-plan.git ../builder && cd ../builder && go install ./cmd/plan && cd $CF_PAGES_REPO_PATH && ~/go/bin/plan build
        ```
    *   **Build output directory**: `public`
    *   **Root directory**: `/` (Leave as default)

4.  **Environment Variables (Optional but Recommended)**:
    *   Ensure `GO_VERSION` is set to `1.22` or a newer version (e.g., `1.22`). This ensures Cloudflare uses a compatible Go version for building the `plan` tool.

Now, every time you `git push` changes to your `my-plan-repo` repository, Cloudflare Pages will automatically fetch the latest builder, build your site, and deploy it.

## How it Works

1.  **Current State**: The builder renders your current `plan.md` to `public/index.html`.
2.  **History**: It walks through the `git log` of your `plan.md`.
3.  **Reconstruction**: For every date the file changed, it retrieves the content from that specific commit.
4.  **Generation**: It generates a static page for that date (e.g., `public/2025/12/01/index.html`) and builds index pages for years and months.

## Requirements

*   **Go**: 1.22+ (to build the tool).
*   **Git**: Must be installed and available in your `$PATH`.