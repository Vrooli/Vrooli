# CLI Commands

The installed command is `tidiness-manager`.

## scan

```bash
tidiness-manager scan <scenario> --type tidiness
tidiness-manager scan <scenario-or-path> --type light --timeout 120
tidiness-manager scan <scenario> --type smart --file api/main.go --campaign-id 123
```

`tidiness` is the default and returns maintainability findings. `light` is retained for metrics compatibility. `smart` requires explicit files.

## issues

```bash
tidiness-manager issues list <scenario> --limit 20
tidiness-manager issues resolve <issue-id> --notes "Split handler"
tidiness-manager issues ignore <issue-id> --notes "Generated fixture"
tidiness-manager issues reopen <issue-id>
```

Filters include scenario, file, folder, category, severity, status, and limit.

## recommend-refactors

```bash
tidiness-manager recommend-refactors <scenario> --limit 10
tidiness-manager recommend-refactors <scenario> --sort-by complexity --min-lines 200
```

Recommendations combine stored metrics and visit context when available.

## scenarios

```bash
tidiness-manager scenarios list
tidiness-manager scenarios get <scenario>
```

Shows scenario-level summaries and detail data for dashboard-style inspection.

## score

```bash
tidiness-manager score <scenario>
```

Shows the aggregate tidiness score for a scenario.

## campaigns

```bash
tidiness-manager campaigns list
tidiness-manager campaigns get <campaign-id>
tidiness-manager campaigns start <scenario> --max-sessions 10 --max-files 5
tidiness-manager campaigns pause <scenario>
tidiness-manager campaigns resume <scenario>
tidiness-manager campaigns stop <scenario>
```

Campaign commands manage auto-tidiness sessions and support scenario-name or ID based actions where implemented.

## tracking

```bash
tidiness-manager tracking visit <file> --scenario <scenario> --note "Refactored"
tidiness-manager tracking exclude <file> --scenario <scenario> --reason "Generated"
tidiness-manager tracking campaign-note --scenario <scenario> --note "Handoff"
```

Tracking wraps visited-tracker workflows used during refactoring campaigns.
