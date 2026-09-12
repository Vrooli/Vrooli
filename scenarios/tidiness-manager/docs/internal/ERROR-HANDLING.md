# Error Handling

## Principles

- Validate user input before expensive work.
- Return sanitized errors to clients.
- Preserve detailed error context in server logs.
- Treat optional analyzer failures as degraded scan metadata when the base scan can still complete.

## API Errors

Bad scenario names, unsafe paths, invalid JSON, unknown fields, unsupported scan types, and invalid limits should return client errors. Database or scanner failures should return safe messages with server-side details in logs.

## CLI Errors

CLI commands should surface the failed operation and the next useful command. JSON output should remain parseable when supported.

## Campaign Errors

Campaign failures should update campaign state with an error reason. Pause, resume, and terminate actions should be explicit and auditable.
