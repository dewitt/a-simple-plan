# A Simple Plan (Builder)

This is the builder application for "A Simple Plan" blog. It is designed to work with a separate "Plan Data" repository.

## Installation

```bash
git clone git@github.com:dewitt/a-simple-plan.git
cd a-simple-plan
go install ./cmd/plan
```

## Usage

Run the `plan` command, pointing it to your plan repository or specific plan file.

```bash
plan build -f /path/to/your/plan-repo/plan.md
```

If you are inside your plan repo:

```bash
plan build
```

## Commands

*   `preview`: Build and serve locally.
*   `build`: Generate static HTML.
*   `save`: Commit local changes (in the plan repo).
*   `publish`: Commit and push to origin (in the plan repo).
*   `revert`: Discard local changes.
*   `rollback`: Revert to a previous version.
*   `edit`: Open the plan file in your editor.

## Plan Repository Structure

Your plan repository should look like this:

```
my-plan-repo/
├── plan.md          (Required) The content of your plan.
├── settings.json    (Optional) Configuration for your site.
└── template.html    (Optional) Custom HTML template.
```

### settings.json

```json
{
  "username": "yourusername",
  "name": "Your Full Name",
  "directory": "/home/yourname",
  "shell": "/bin/zsh",
  "timezone": "America/New_York",
  "title": "My Plan"
}
```

### template.html

If you provide a custom template, use the following placeholders:
*   `{{content}}`: The rendered markdown content.
*   `{{onSince}}`: Creation date string.
*   `{{modTimeUnix}}`: Modification timestamp.
*   `{{username}}`, `{{fullname}}`, `{{directory}}`, `{{shell}}`, `{{title}}`: From settings.
