# Testing

## Focused Validation

```bash
test-genie execute tidiness-manager --phases quality --json
test-genie execute tidiness-manager --phases tidiness --json
test-genie execute tidiness-manager --phases docs --json
```

## Surface Tests

```bash
cd scenarios/tidiness-manager/api && GOWORK=off go test ./...
cd scenarios/tidiness-manager/cli && GOWORK=off go test ./...
cd scenarios/tidiness-manager/ui && pnpm test
```

## Quality Gates

```bash
cd scenarios/tidiness-manager/api && GOWORK=off golangci-lint run ./...
cd scenarios/tidiness-manager/cli && GOWORK=off golangci-lint run ./...
cd scenarios/tidiness-manager/ui && pnpm run lint
cd scenarios/tidiness-manager/ui && pnpm run type-check
```

## Full Scenario Test

```bash
vrooli scenario test tidiness-manager
```

Use the printed wait command once if the durable run backgrounds.
