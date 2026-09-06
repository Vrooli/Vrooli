acceptance_allow:
  - scenarios/web-console/**
  - packages/proto/schemas/web-console/**
  - packages/proto/gen/**/web-console/**
  - scenarios/agent-manager/**
  - packages/proto/schemas/agent-manager/**
  - packages/proto/gen/**/agent-manager/**
  - scenarios/search-hub/**
  - docs/architecture/web-console-conversation-continuity.md
acceptance_deny:
  - .vrooli/dependencies/approved-dependencies.json
  - scenarios/plan-manager/**
  - scenarios/git-control-tower/**
  - scenarios/test-genie/**
  - scenarios/prompt-manager/**

The Agent Manager and Search Hub allowances are integration-only: extend their published source/search contracts and fixtures only where their owning active plan does not already do so. Web Console lifecycle authority MUST NOT migrate into those scenarios. Shared substrate changes are outside the initial boundary; if investigation proves one necessary, extend the boundary through a logged Plan Manager decision rather than writing a workaround. Runtime state beneath `/home/matthalloran8/.vrooli/state/**` is evidence and migration input, not source-controlled implementation: never mutate it before backup/restore proof and the explicit reconciliation phase.
