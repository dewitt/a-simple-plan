# Design Document

## Goals
To create a publishing workflow that is as simple as maintaining a single text file, yet robust enough to produce a full-featured blog with history.

## Architecture

The system consists of two distinct components:
1.  **The Builder (`a-simple-plan`)**: The logic and tooling.
2.  **The Data (`my-plan-repo`)**: The content and configuration.

### The Builder
*   **Language**: Go.
*   **Dependencies**: `git` (CLI), `goldmark` (Markdown), `chroma` (Syntax Highlighting).
*   **Responsibility**: 
    *   Parse the plan directory.
    *   Read configuration.
    *   Interact with `git` to retrieve current and historical file versions.
    *   Render Markdown to HTML.
    *   Generate the directory structure in `public/`.

### The Data
*   **Format**: Standard Git Repository.
*   **Files**:
    *   `plan.md`: The single source of content.
    *   `settings.json`: Configuration (User details, title).
    *   `template.html`: Custom HTML layout.

## Data Flow

1.  **Initialization**:
    *   User runs `plan build`.
    *   System identifies the context (Plan Directory).
    *   Config and Template are loaded.

2.  **Current Build**:
    *   `plan.md` is read from the filesystem.
    *   Metadata (mod time) is gathered.
    *   Content is rendered and written to `public/index.html`.

3.  **History Build**:
    *   System runs `git log` on `plan.md`.
    *   Unique dates are identified.
    *   For each date, the latest commit hash is found.
    *   `git show` retrieves the file content for that hash.
    *   Historical content is rendered to `public/YYYY/MM/DD/index.html`.
    *   Index pages are generated for `public/YYYY/` and `public/YYYY/MM/`.

## URL Structure

*   `/`: The current version of the plan.
*   `/YYYY/`: List of updates in that year.
*   `/YYYY/MM/`: List of updates in that month.
*   `/YYYY/MM/DD/`: The specific version of the plan as it existed on that day.