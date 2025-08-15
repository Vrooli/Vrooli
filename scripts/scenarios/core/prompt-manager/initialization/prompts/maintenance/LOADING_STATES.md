## 🚀 Purpose & Scope

* **Task ID:** `LOADING_STATES`
* **Goal:** Ensure every async path shows a clear loading indicator → progress → either data, an empty state, or an error fallback.
* **Out-of-scope:** Major visual redesigns; keep changes mechanical and incremental.

---

## 1  Hard Rules

| Rule                                                               | Why                                                              |
| ------------------------------------------------------------------ | ---------------------------------------------------------------- |
| Don’t run Git commands                                             | Human reviews and commits.                                       |
| Don’t add new dependencies without approval                        | Prevents bundle bloat.                                           |
| Preserve `.js` extension in TS imports                             | Node ESM.                                                        |
| Loading UIs **must** be accessible (`aria-busy`, focus management) | Screen-reader users deserve parity. ([developer.mozilla.org][1]) |

---

## 2  Key Concepts

### 2.1  Loading Indicators

* **Spinner / Indeterminate** – use when duration is unknown; avoid for waits > 10 s without context. ([uxplanet.org][2])
* **Progress bar / Determinate** – show % when you can calculate it. ([m3.material.io][3], [balsamiq.com][4])
* **Skeleton screen** – placeholder layout that reduces perceived latency. ([nngroup.com][5], [uxdesign.cc][6])

### 2.2  Empty vs Error

* Empty (no data) should explain next steps, not just say “0 results”. ([carbondesignsystem.com][7], [design-system.agriculture.gov.au][8])
* Error fallback must offer retry or support contact and be logged. ([design-system.agriculture.gov.au][8])

### 2.3  Optimistic & Suspense Patterns

* Optimistic UI instantly updates the view then reconciles with the server to feel “instant”. ([simonhearne.com][9], [medium.com][10])
* React 18 `Suspense`/`lazy` lets you wrap components and supply a `<fallback>` loader. ([react.dev][11], [react.dev][12])

---

## 3  Detection & Tooling

| Target                                      | Command                                                       | Note                                         |            |            |
| ------------------------------------------- | ------------------------------------------------------------- | -------------------------------------------- | ---------- | ---------- |
| Components missing fallback in Suspense     | `rg "<Suspense[^>]*fallback={}" -l packages/ui`               | Empty fallback = bad UX.                     |            |            |
| No skeleton/loader but network call present | \`rg "useSWR\\(                                               | axios\\.get" --files-without-match "Skeleton | Spinner"\` | Heuristic. |
| Missing `aria-busy`                         | `pnpm lint --rule 'jsx-a11y/aria-busy-has-live-region:error'` | Custom ESLint rule.                          |            |            |
| Hard-coded “Loading…” text                  | `rg "\"Loading\\.\\.\\.\""`                                   | Replace with component for a11y.             |            |            |
| Components without error boundary           | Check for `try/catch` or `<ErrorBoundary>` sibling.           |                                              |            |            |

Prioritise files with no `AI_CHECK` or oldest `LAST` date for `LOADING_STATES`.

---

## 4  Implementation Patterns

### 4.1  Spinners & Progress Bars

```tsx
import { CircularProgress, LinearProgress } from "@mui/material";

export function PageLoader() {
  return (
    <section role="status" aria-busy="true">
      <CircularProgress aria-label="Loading content" />
    </section>
  );
}
```

* Wrap spinner in an element that sets `aria-busy`. ([developer.mozilla.org][1])
* For known durations, swap to `LinearProgress variant="determinate" value={pct}`. ([m3.material.io][3])

### 4.2  Skeleton Screens

```tsx
import Skeleton from "react-loading-skeleton";

export const CardSkeleton = () => (
  <article aria-busy="true">
    <Skeleton height={200} />
    <Skeleton count={3} />
  </article>
);
```

* Replace content nodes directly so layout shift is minimal. ([skeletonreact.com][13], [uxdesign.cc][6])

### 4.3  Error & Empty States

```tsx
function ErrorState({ retry }: { retry(): void }) {
  return (
    <div role="alert">
      <p>We couldn’t load your dashboard.</p>
      <Button onClick={retry}>Try again</Button>
    </div>
  );
}
```

* Provide at least one **recovery action** (retry, refresh, contact). ([carbondesignsystem.com][7])
* Log errors with context for back-end observability.

---

## 5  Step-by-Step Workflow

1. **Scan** with detection queries.

2. **Add or fix** loading, skeleton, or error components using §4 snippets.

3. **Wire** them via state flags (`isLoading`, `isError`, `data.length === 0`).

4. **Verify**

   * `pnpm test` (React Testing Library: `expect(screen.getByRole('status')).toBeInTheDocument()`),
   * `pnpm tsc --noEmit`,
   * `pnpm lint`.

5. **Update AI\_CHECK** at top of each touched file:

   ```ts
   // AI_CHECK: LOADING_STATES=<incrementedCount> | LAST: 2025-06-16
   ```

6. **Stdout** one-liner per file: `src/components/UserList.tsx → added skeleton + error state`.

---

## 6  Fast Checklist

| ✔︎                                                     | Item |
| ------------------------------------------------------ | ---- |
| Loading UI appears <200 ms after action starts.        |      |
| Proper ARIA attributes (`role="status"`, `aria-busy`). |      |
| Skeleton replaces actual nodes, no big CLS.            |      |
| Error fallback offers a retry path.                    |      |
| Tests cover success, loading, and error branches.      |      |
| `AI_CHECK` updated only for `LOADING_STATES`.          |      |

---

## 7  Reference Links

1. UX Design – Loading & Progress Indicators series ([uxdesign.cc][14])
2. NN/g – Skeleton Screens 101 ([nngroup.com][5])
3. MDN – `aria-busy` ([developer.mozilla.org][1])
4. React Docs – `<Suspense>` fallback ([react.dev][11])
5. Material 3 – Progress Indicators ([m3.material.io][3])
6. React Skeleton library ([skeletonreact.com][13])
7. Optimistic UI patterns blog ([simonhearne.com][9])
8. UX Planet – Progress Indicators best practices ([uxplanet.org][2])
9. UX Collective – Skeleton screen dos & don’ts ([uxdesign.cc][6])
10. Carbon Design – Empty States pattern ([carbondesignsystem.com][7])
11. Agriculture DS – Loading, Empty & Error states ([design-system.agriculture.gov.au][8])
12. DockYard – Accessible Loading Indicators ([dockyard.com][15])
13. React Docs – `lazy` & Suspense pattern ([react.dev][12])

---

### Remember

Users forgive waiting when **you** keep them informed, entertained, and in control. Nail these states, and the whole product will feel faster and friendlier—no extra milliseconds required.

[1]: https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Reference/Attributes/aria-busy?utm_source=chatgpt.com "ARIA: aria-busy attribute - MDN Web Docs - Mozilla"
[2]: https://uxplanet.org/progress-indicators-4-common-styles-91a12b86060c?utm_source=chatgpt.com "Progress Indicators: 4 Common Styles | by Nick Babich | UX Planet"
[3]: https://m3.material.io/components/progress-indicators/guidelines?utm_source=chatgpt.com "Progress indicators – Material Design 3"
[4]: https://balsamiq.com/learn/ui-control-guidelines/progress-bars/?utm_source=chatgpt.com "Progress bar guidelines - Balsamiq"
[5]: https://www.nngroup.com/articles/skeleton-screens/?utm_source=chatgpt.com "Skeleton Screens 101 - NN/g"
[6]: https://uxdesign.cc/what-you-should-know-about-skeleton-screens-a820c45a571a?utm_source=chatgpt.com "Everything you need to know about skeleton screens - UX Collective"
[7]: https://carbondesignsystem.com/patterns/empty-states-pattern/?utm_source=chatgpt.com "Empty states - Carbon Design System"
[8]: https://design-system.agriculture.gov.au/patterns/loading-error-empty-states?utm_source=chatgpt.com "Loading, empty and error states pattern | Agriculture Design System"
[9]: https://simonhearne.com/2021/optimistic-ui-patterns/?utm_source=chatgpt.com "Optimistic UI Patterns for Improved Perceived Performance"
[10]: https://medium.com/%40kyledeguzmanx/what-are-optimistic-updates-483662c3e171?utm_source=chatgpt.com "What Are Optimistic Updates? - Medium"
[11]: https://react.dev/reference/react/Suspense?utm_source=chatgpt.com "<Suspense> – React"
[12]: https://react.dev/reference/react/lazy?utm_source=chatgpt.com "lazy - React"
[13]: https://skeletonreact.com/?utm_source=chatgpt.com "React Skeleton - Create Content Loader"
[14]: https://uxdesign.cc/loading-progress-indicators-ui-components-series-f4b1fc35339a?utm_source=chatgpt.com "Loading & progress indicators — UI Components series"
[15]: https://dockyard.com/blog/2020/03/02/accessible-loading-indicatorswith-no-extra-elements?utm_source=chatgpt.com "Accessible Loading Indicators—with No Extra Elements! - DockYard"
