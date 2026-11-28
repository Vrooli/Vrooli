# Troubleshooting

- If the API does not start, run `make logs` and check schema initialization before inspecting the UI.
- If position is empty, configure/select a book and accounts. Empty is undefined, not zero.
- If position is partial, inspect the adapter id, reason, and last-success time. Re-run the adapter after its source recovers.
- If an event is duplicated, confirm the adapter id and external id; the original posting is returned intentionally.
- If a correction is needed, use `journal-reverse`. There is no edit or delete path.
- If the UI has stale generated strings or proto types, run `make setup` through the scenario lifecycle, then `pnpm type-check` and `pnpm build` from `ui/`.
