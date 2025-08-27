## 🚀 Purpose

*Task ID `REACT_PERF`* is for **detecting, measuring, and fixing React-18 performance bottlenecks** without changing behaviour or breaking repo rules.

---

## ⚙️ Hard Rules (inherits `CLAUDE.md`)

| Rule                           | Why                         |
| ------------------------------ | --------------------------- |
| No Git commands                | Human review & commit gate. |
| Prefer removals over additions | Slimmer JS ⇒ faster JS.     |
| No new prod deps ≥ minor       | Stability first.            |
| Keep `.js` import extensions   | Node ESM compliance.        |

---

## 1 Identification Cheat-Sheet (React 18)

| Smell                          | How to Detect                                                                                 | Fix Strategy                                                                                         |
| ------------------------------ | --------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| **Over-recreated callbacks**   | DevTools Profiler or **why-did-you-render** logs ([reddit.com][1], [blog.logrocket.com][2])   | Wrap in `useCallback` only when a memoised child needs a stable ref; otherwise remove. ([dev.to][3]) |
| **Everything in `useEffect`**  | `eslint-plugin-react-hooks` warnings; grep for `useEffect(` ([github.com][4], [npmjs.com][5]) | Derive data in render; keep effects for I/O; split multi-concern effects. ([github.com][4])          |
| **Context cascades**           | DevTools flood of subtree re-renders; search `createContext(` ([github.com][6], [dev.to][3])  | Move hot state to selector-based **Zustand** stores. ([reddit.com][1])                               |
| **Heavy MUI bundle**           | `npm run analyze` (webpack/vite visualizer) shows large `@mui/*` chunk ([codelevate.com][7])  | Replace leaf widgets with Tailwind utilities or headless primitives. ([codelevate.com][7])           |
| **Slow long lists**            | Profiler shows commits >16 ms                                                                 | Virtualise via `react-window` or `react-virtual`. ([patterns.dev][8])                                |
| **Blocking updates**           | Missing `startTransition` guards ([react.dev][9])                                             | Wrap low-priority state bursts in `startTransition`. ([react.dev][9], [github.com][10])              |
| **Client-only fetch on mount** | `useEffect(fetch …)` in page shells                                                           | Shift data to streaming-SSR or framework loaders. ([patterns.dev][8])                                |
| **No code-splitting**          | Bundle analyser shows monolith; no `React.lazy` imports                                       | Split with `lazy` + `Suspense`; add route-level boundaries. ([codelevate.com][7])                    |

---

## 2 Dual Toolkits

### 2.1 Toolkit A – **Browser Automation Available**

*Use when you have the **`puppeteer` MCP tool**.*

1. **Launch full stack**

   ```bash
   vrooli develop --target docker --detached yes     # brings up app on http://localhost:3000
   ```

2. **Measure baseline**

   ```ts
   import puppeteer from 'puppeteer';
   import lighthouse from 'lighthouse';

   const {browser, page} = await puppeteer.launch({headless: true});
   await page.goto('http://localhost:3000');

   // Lighthouse
   const lh = await lighthouse('http://localhost:3000', {port: (new URL(browser.wsEndpoint())).port});
   await page.tracing.start({path: 'trace.json'});
   // interact if needed...
   await page.tracing.stop();
   await browser.close();
   ```

   *See Puppeteer-Lighthouse guide* ([github.com][11]).

3. **React Profiler export** – use DevTools hook to save flame-chart JSON. ([github.com][10], [react.dev][12])

4. **Apply fix → rebuild containers (`docker compose restart ui`) → re-measure.**

### 2.2 Toolkit B – **No Browser Access**

* Run the same `develop.sh` script so backend APIs resolve for component tests.
* Rely on:

  * **Static analysis** (`eslint-plugin-react-hooks`, bundle analyser). ([npmjs.com][5], [codelevate.com][7])
  * **why-did-you-render** console output. ([github.com][6])
  * **Synthetic render benchmarks** (Jest + `performance.now()`).
  * **`size-limit`** to guard bundle size.

---

## 3 Step-by-Step Procedure

1. **Select targets**

   ```bash
   git diff --name-only origin/main...HEAD | rg '\.tsx?$' \
     | xargs rg -L "AI_CHECK:.*REACT_PERF"
   ```

2. **Baseline profile** (Toolkit A = Lighthouse/trace; Toolkit B = Profiler stats).

3. **Apply a single fix class** (§1 table).

4. **Re-measure** – expect ≥ 15 % faster render OR smaller bundle.

5. **Run tests & lints** (`pnpm test && pnpm lint`).

6. **Update comment**

   ```ts
   // AI_CHECK: REACT_PERF=<incrementedCount> | LAST: 2025-06-16
   ```

7. **Stdout summary**: `Component → toolkit → issue → fix → Δmetric`.

---

## 4 Fast Checklist

| ✔︎                                                                        | Item |
| ------------------------------------------------------------------------- | ---- |
| Effects trimmed; state derived in render.                                 |      |
| Zustand selector stores replace noisy Context.                            |      |
| Tailwind replaces heavy MUI where feasible.                               |      |
| Lists virtualised; code-split rare routes.                                |      |
| `startTransition` wraps long-running updates.                             |      |
| Toolkit A: Lighthouse JSON & trace saved; Toolkit B: Profiler file saved. |      |
| Tests & lints pass.                                                       |      |

---

## 5 Anti-Patterns

| ❌                                       | Why Bad                                                 |
| --------------------------------------- | ------------------------------------------------------- |
| Blanket `useCallback` everywhere        | Adds memory/indirection; no perf gain. ([dev.to][3])    |
| Giant `useEffect` rerunning each render | Network loops & layout jank. ([github.com][4])          |
| High-freq state in Context              | Whole-tree re-renders; use Zustand. ([reddit.com][1])   |
| MUI in hot grid paths                   | Runtime CSS-in-JS + big bundle. ([codelevate.com][7])   |
| Client fetch for static data            | Delays FCP; shift to SSR/streaming. ([patterns.dev][8]) |

---

## 6 Reference Resources

1. Puppeteer + Lighthouse automation guide ([github.com][11])
2. React DevTools programmatic profiler export request ([github.com][10])
3. `<Profiler>` API docs ([react.dev][12])
4. **why-did-you-render** GitHub ([github.com][6])
5. ESLint Hooks plugin exhaustive-deps docs ([npmjs.com][5])
6. React 18 release post (automatic batching, `startTransition`) ([react.dev][9])
7. React 18 concurrency article (transitions) ([github.com][10])
8. Zustand vs Context performance discussion ([dev.to][3])
9. Reddit thread on Zustand adoption ([reddit.com][1])
10. Tailwind vs MUI bundle-size study 2024 ([codelevate.com][7])
11. List virtualization primer (react-window) ([patterns.dev][8])
12. LogRocket guide on **why-did-you-render** debugging ([blog.logrocket.com][2])
13. Discussion on disabling exhaustive-deps when using a compiler ([github.com][13])

---

### Remember

> **Launch full app first (`vrooli develop …`)**—everything else builds on that.
> **Measure ➜ Fix ➜ Measure**—never “optimise” blind.

[1]: https://www.reddit.com/r/react/comments/1g6ci6n/when_to_use_store_zustand_vs_context_vs_redux/?utm_source=chatgpt.com "When to use Store (Zustand) vs Context vs Redux ? : r/react - Reddit"
[2]: https://blog.logrocket.com/debugging-react-performance-issues-with-why-did-you-render/?utm_source=chatgpt.com "Debugging React performance issues with Why Did You Render"
[3]: https://dev.to/shubhamtiwari909/react-context-api-vs-zustand-pki?utm_source=chatgpt.com "React | Context API vs Zustand - DEV Community"
[4]: https://github.com/GoogleChrome/lighthouse/issues/5518?utm_source=chatgpt.com "Running Lighthouse and Puppeteer on Headless Chrome on RHLS ..."
[5]: https://www.npmjs.com/package/eslint-plugin-react-hooks?utm_source=chatgpt.com "eslint-plugin-react-hooks - NPM"
[6]: https://github.com/welldone-software/why-did-you-render?utm_source=chatgpt.com "welldone-software/why-did-you-render - GitHub"
[7]: https://www.codelevate.com/blog/tailwind-css-vs-material-ui-a-comprehensive-guide-2024?utm_source=chatgpt.com "Tailwind CSS vs Material UI - A comprehensive guide 2024"
[8]: https://www.patterns.dev/vanilla/virtual-lists/?utm_source=chatgpt.com "List Virtualization - Patterns.dev"
[9]: https://react.dev/blog/2022/03/29/react-v18?utm_source=chatgpt.com "React v18.0"
[10]: https://github.com/facebook/react/issues/16744?utm_source=chatgpt.com "Devtools: Allow saving and loading a profiler run as JSON ... - GitHub"
[11]: https://github.com/GoogleChrome/lighthouse/blob/main/docs/puppeteer.md?utm_source=chatgpt.com "lighthouse/docs/puppeteer.md at main - GitHub"
[12]: https://react.dev/reference/react/Profiler?utm_source=chatgpt.com "<Profiler> – React"
[13]: https://github.com/reactwg/react-compiler/discussions/18?utm_source=chatgpt.com "Using `eslint-plugin-react-hooks` together with `eslint ... - GitHub"
