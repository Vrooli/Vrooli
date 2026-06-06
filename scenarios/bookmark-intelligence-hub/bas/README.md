# Bookmark Intelligence Hub BAS Registry

This folder holds behavior acceptance scenarios aligned to the PRD operational targets.

Structure:

- `cases/<target>/<surface>/`: target-specific checks by surface, such as `api`, `cli`, or `ui`.
- `flows/`: cross-target journey notes and readiness checks that are not yet automated end-to-end flows.

The current readiness flow files cover future surfaces referenced by requirements that do not yet have dedicated application code:

- `flows/browser-extension-readiness.md`
- `flows/mobile-review-readiness.md`
- `flows/calendar-scheduling-readiness.md`
- `flows/team-collaboration-readiness.md`

Add new BAS files under the matching PRD target folder when implementing a concrete workflow. Keep selectors centralized in `ui/src/consts/selectors.ts` if the UI is migrated to the React/Vite structure.
