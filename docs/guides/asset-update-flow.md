# Asset update flow

Released React Component Library versions are immutable. Cross-cutting edits
must use the catalog draft lane so the existing release remains reproducible.

## One asset

Open a patch draft, edit only the draft directory, then promote it after the
focused checks pass:

```bash
catalog draft open react-component-library:Button --bump patch
# edit library/components/Button/versions/<draft>/Button.tsx and companions
catalog draft promote react-component-library:Button \
  --changelog "Normalize the shared button interaction"
```

To abandon the draft, use `catalog draft discard`:

```bash
catalog draft discard react-component-library:Button
```

The commands refuse an attempt to write a released version. Create a new
draft with `catalog draft open` and make the change there.

## Bulk migration

Use `--all` with an optional substring selector to open drafts for a mechanical
change across the catalog. Review the generated versions, apply the transform
to each draft directory, and promote each reviewed draft:

```bash
catalog draft open --all --match "form" --bump patch
# review and transform each selected draft
catalog draft promote --all --changelog "Adopt the form validation contract"
```

The bulk command is intentionally explicit: opening, editing, and promoting
are separate steps, and an interrupted migration can be safely discarded.
