# Connecting Skills to References

## Prerequisites

- DTV API running (`make start` or `vrooli scenario start development-toolchain-validator`)
- prompt-manager API running (DTV reads skills from it)
- At least one reference scenario registered

## Step 1: Register a Reference Scenario

Before connecting skills, register the reference scenario:

```bash
development-toolchain-validator references add reference-react-vite --template react-vite
```

Verify registration:
```bash
development-toolchain-validator references list
```

## Step 2: Connect a Skill

Connect a prompt-manager steer skill to the reference:

```bash
development-toolchain-validator skills connect api-steer --reference reference-react-vite
```

This fetches the skill's current version and content hash from prompt-manager's API and stores them with the connection. The connection starts as **unconfigured** — no structural expectations or CLI assertions are defined yet.

### Connecting Multiple Skills

Connect all applicable skills for a reference:

```bash
# Core architectural steers
development-toolchain-validator skills connect api-steer --reference reference-react-vite
development-toolchain-validator skills connect storage-steer --reference reference-react-vite
development-toolchain-validator skills connect cli-steer --reference reference-react-vite
development-toolchain-validator skills connect interoperability-steer --reference reference-react-vite
development-toolchain-validator skills connect unit-testing-architecture-steer --reference reference-react-vite

# Quality steers
development-toolchain-validator skills connect documentation-health --reference reference-react-vite
development-toolchain-validator skills connect screaming-architecture-audit --reference reference-react-vite
development-toolchain-validator skills connect react-coherence --reference reference-react-vite
```

### Listing Connections

```bash
# All connections for a reference
development-toolchain-validator skills list --reference reference-react-vite

# Show connection details including version and drift status
development-toolchain-validator skills show api-steer --reference reference-react-vite
```

## Step 3: Add Expectations

This is where value comes from. Each expectation makes the connection more concrete and testable.

See [Writing Assertions](writing-assertions.md) for detailed syntax.

### Quick Example: api-steer

```bash
# Expect domain-organized handler directories
development-toolchain-validator expectations add structural \
  --connection api-steer:reference-react-vite \
  --type folder --path "api/handlers/projects/" --required true \
  --description "Projects domain module"

development-toolchain-validator expectations add structural \
  --connection api-steer:reference-react-vite \
  --type folder --path "api/handlers/tasks/" --required true \
  --description "Tasks domain module"

# Expect no single monolithic routes file
development-toolchain-validator expectations add structural \
  --connection api-steer:reference-react-vite \
  --type file --path "api/routes.go" --required false \
  --description "Should NOT have a single monolithic routes file"

# Expect auditor passes all API rules
development-toolchain-validator expectations add cli-tool \
  --connection api-steer:reference-react-vite \
  --command "scenario-auditor audit reference-react-vite --json" \
  --path "$.bySeverity.critical" --op eq --value 0
```

## Step 4: Validate

Run validation against the reference:

```bash
development-toolchain-validator validate reference-react-vite
```

This runs all configured expectations for all connected skills and produces a comprehensive report.

## Step 5: Handle Drift

Over time, skills evolve in prompt-manager. Check for drift:

```bash
development-toolchain-validator drift check --reference reference-react-vite
```

When drift is detected, review the skill's changes and update expectations if needed:

```bash
# See what changed
development-toolchain-validator drift show api-steer --reference reference-react-vite

# Update the version pin after reviewing
development-toolchain-validator skills refresh api-steer --reference reference-react-vite
```

## Best Practices

1. **Start unconfigured**: Connect all applicable skills first, then add expectations incrementally. Unconfigured connections are valuable signals.

2. **Focus on structural contracts**: Define expectations for things the skill clearly mandates (folder structure, file patterns), not implementation details.

3. **Use CLI assertions for tool output**: If the skill references a CLI tool for validation (e.g., "run `scenario-auditor audit`"), create a CLI assertion for it.

4. **Review drift promptly**: When drift is detected, the expectations may be stale. Review the skill's changes before assuming the expectations are still correct.

5. **Document expectations**: Use the `--description` flag to explain why each expectation exists. Future maintainers need to understand the reasoning.
