# Auto Steer Profile Registry

This folder is the **authoritative registry** for Auto Steer profiles. Profiles are stored as JSON files and indexed by `metadata.json` so they can be versioned in git and read by agents without database access.

## Structure
```
profiles/
  metadata.json
  <profile-id>/
    profile.json
    README.md (optional)
```

## Rules
- `metadata.json` is authoritative for **id, name, description, tags, kind, file path**.
- `profile.json` contains the full Auto Steer profile configuration (phases, quality gates, etc.). Phases store `skill_id`, `skill_name`, and `modes` (breadcrumb hints) instead of a single mode string.
- Keep `metadata.json` and `profile.json` in sync; mismatches are treated as errors.
- `kind` is either `template` or `profile`.

## API behavior
The ecosystem-manager API reads and writes these files directly:
- `POST /api/auto-steer/profiles` creates a new profile file and metadata entry.
- `PUT /api/auto-steer/profiles/{id}` updates the profile file and metadata entry.
- `DELETE /api/auto-steer/profiles/{id}` removes the profile file and metadata entry.

Templates live in the same registry with `kind: "template"` and are returned by:
- `GET /api/auto-steer/templates`

## Notes
- Execution history and runtime state live in the embedded SQLite store (`api-core/storage`), not in the git-tracked profile files.
- Profile IDs must remain stable (tasks reference `auto_steer_profile_id`).
