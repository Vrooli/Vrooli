/**
 * React Router v7 future flags, opted into now so v6 doesn't emit migration
 * `console.warn`s (which the strict test console-guard turns into failures) and
 * so the eventual v7 bump is a no-op. `v7_startTransition` wraps router state
 * updates in `React.startTransition`; `v7_relativeSplatPath` fixes relative
 * paths under splat routes.
 *
 * This lives in its own dependency-free module (NOT in routes.tsx) so the test
 * render harness can import the flags without pulling the whole route table —
 * and the pages/features it references — into every test's module graph, which
 * would create an import cycle that breaks `vi.mock` hoisting.
 */
export const routerFutureFlags = {
  v7_startTransition: true,
  v7_relativeSplatPath: true,
} as const;
