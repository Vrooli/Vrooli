# Driver Test Helpers

This directory is the Playwright-driver testutil layer. It is test-only infrastructure and is protected by `tests/unit/boundaries/no-prod-testutil-imports.test.ts`; production files under `src/` must not import from here.

## Responsibilities

- `playwright-mocks.ts`: Playwright object fakes for `Page`, `Browser`, `BrowserContext`, `Frame`, request/response objects, and recording initialization.
- `http-mocks.ts`: Node HTTP request/response harnesses for route and body-parser tests.
- `instruction-factory.ts`: typed instruction/action builders. Prefer `createTypedInstruction` for new handler tests; `createTestInstruction` remains for compatibility with legacy plain-params paths.
- `test-config.ts`: deterministic driver config fixture with deep partial overrides.
- `index.ts`: compatibility barrel for existing tests.

## Usage Rules

- Keep fakes small and deterministic by default.
- Add a new helper file only when a seam recurs across tests.
- Keep real-browser behavior in integration or selector tests; unit helpers should stay mock-based.
- Prefer extending typed builders over copying proto construction into test files.
