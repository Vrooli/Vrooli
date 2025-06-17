## 🗝️ Key Take-aways *(one-paragraph overview)*

Robust error handling in a REST backend means you **throw typed `Error` subclasses, log through the shared Winston JSON logger, fail fast on programming bugs, retry only transient faults with exponential back-off, and return clear HTTP status codes with machine-readable bodies**. Follow the guidelines below and your services will surface actionable traces, keep noisy stack leaks out of user responses, and stay resilient under real-world load. ([medium.com][1], [betterstack.com][2], [expressjs.com][3], [docs.bullmq.io][4], [npmjs.com][5], [dev.to][6], [answers.netlify.com][7], [stackoverflow.com][8], [stackoverflow.com][9], [stackoverflow.com][10])

---

## 1 Purpose & Scope

| Field            | Value                                                                                                                |
| ---------------- | -------------------------------------------------------------------------------------------------------------------- |
| **Task ID**      | `ERROR_HANDLING`                                                                                                     |
| **Goal**         | Replace ad-hoc prints with **typed errors + Winston logging + sane retries** in the REST API and background workers. |
| **Out of scope** | Alerting/monitoring dashboards (see `MONITORING`).                                                                   |

---

## 2 Hard Rules

| Rule                                                                                                 | Why                                                                |
| ---------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| **Always use the shared `logger` (Winston)**—no `console.*` outside tests.                           | Ensures JSON output & file/console routing. ([betterstack.com][2]) |
| **Throw subclasses of `Error`, never strings/numbers.**                                              | Keeps stack & prototype intact. ([medium.com][1])                  |
| **Crash fast in dev/test** by running Node with `--unhandled-rejections=strict`.                     | Forces you to catch async errors. ([answers.netlify.com][7])       |
| **Respond with the right HTTP code** (`400/422` user input, `429` rate-limit, `5xx` internal).       | Clear contract for clients. ([expressjs.com][3])                   |
| **Retry only transient faults** (network timeouts, `ECONNRESET`) with exponential back-off + jitter. | Prevents thundering herds. ([npmjs.com][5], [dev.to][6])           |

---

## 3 Winston Logger (server & jobs)

```ts
import { logger } from "../../logger"; // Use accurate relative path

logger.error("Detailed msg", {
  trace: "abcd-EF12",
  error,
  reqId,          // populated by request-ID middleware
  service: "express-server",
});
```

* **JSON output** with `timestamp`, `level`, and all meta. ([stackoverflow.com][8])
* File transports (`combined.log`, `error.log`) rotate at 5 MB unless you’re in tests.
* Console transport enabled in dev for rapid feedback. ([betterstack.com][2])

---

## 4 Error Taxonomy

| Category                    | Example                      | What you do                                                        |
| --------------------------- | ---------------------------- | ------------------------------------------------------------------ |
| **Programmer**              | Null deref, invariant fail   | Throw, log at **error**, add test.                                 |
| **Operational (transient)** | `ETIMEDOUT`, 503 from Stripe | Retry ×3 (500 ms → 2 s) then bubble up. ([npmjs.com][5])           |
| **Business / domain**       | “credits depleted”           | Return structured JSON `{code:"INSUFFICIENT_CREDITS"}` + HTTP 409. |
| **User input**              | Zod validation error         | HTTP 400/422; log at **warn** (not error).                         |

---

## 5 Detection Toolbelt

| Tool                                                                | Command                              | Catches                                                       |
| ------------------------------------------------------------------- | ------------------------------------ | ------------------------------------------------------------- |
| ESLint `no-throw-literal`, `handle-callback-err`                    | `pnpm lint`                          | Non-Error throws, ignored callback err. ([docs.bullmq.io][4]) |
| Node strict rejections                                              | `node --unhandled-rejections=strict` | Async holes. ([github.com][11])                               |
| Winston `logger.exceptions.handle()` / `logger.rejections.handle()` | activated in logger factory          | Logs before exit. ([github.com][12])                          |
| BullMQ `worker.on("failed")`                                        | in queue factory                     | Centralised job failures. ([docs.bullmq.io][4])               |
| Express default error flow                                          | final `app.use(errorHandler)`        | One funnel for REST responses. ([expressjs.com][3])           |

---

## 6 Implementation Patterns

### 6.1 Typed Error Base

```ts
export abstract class AppError extends Error {
  abstract readonly type: 'VALIDATION' | 'DOMAIN' | 'INTERNAL';
  constructor(msg: string, public meta?: Record<string, unknown>) {
    super(msg);
    this.name = new.target.name;
  }
}
```

### 6.2 Async Wrappers

```ts
async function callStripe() {
  try {
    return await stripeClient.charge(...);
  } catch (err) {
    if (isTransient(err)) return retryWithBackoff(callStripe, err);
    throw new InternalError("Stripe failure", { err });
  }
}
```

### 6.3 Express Middleware

```ts
import { Request, Response, NextFunction } from "express";

export function errorHandler(
  err: unknown,
  _req: Request,
  res: Response,
  _next: NextFunction,
) {
  logger.error("Unhandled API error", { err });
  if (err instanceof ValidationError) {
    return res.status(422).json({ code: err.type, details: err.meta });
  }
  res.status(500).json({ code: "INTERNAL_SERVER_ERROR" });
}
```

Add this **after** all other routes. ([expressjs.com][3])

### 6.4 React Error Boundary (for client bundles)

````tsx
class RootBoundary extends React.Component {
  state = { crashed: false };
  static getDerivedStateFromError() { return { crashed: true }; }
  componentDidCatch(error, info) {
    logger.error("UI crash", { error, info });
  }
  render() { return this.state.crashed ? <CrashScreen/> : this.props.children; }
}
``` :contentReference[oaicite:15]{index=15}  

---

## 7 Step-by-Step Workflow  

1. **Scan for smells**

   ```bash
   rg "console\\.error|throw ['\"][^<]|catch \\(e\\) \\{\\}" packages/server packages/jobs
````

2. **Refactor** with §6 patterns.

3. **Add/extend tests**—simulate rejection and assert a `logger.error()` call.

4. **Run** `pnpm test && pnpm lint && pnpm tsc --noEmit`.

5. **Update `AI_CHECK`** at file top:

   ```ts
   // AI_CHECK: ERROR_HANDLING=<incrementedCount> | LAST: 2025-06-17
   ```

6. **Stdout summary** → `jobs/worker.ts → replaced console.error, added retry`.

---

## 8 Fast Checklist

| ✔︎                                               | Item |
| ------------------------------------------------ | ---- |
| Every thrown value extends `Error`.              |      |
| Winston logger used everywhere (no raw console). |      |
| JSON logs include `trace` or request ID.         |      |
| React & BullMQ error funnels installed.          |      |
| Tests + strict compiler flags green.             |      |
| `AI_CHECK` updated only for `ERROR_HANDLING`.    |      |

---

## 9 Reference Links

* Express docs – error-handling middleware. ([expressjs.com][3])
* Medium – TypeScript error patterns for Express. ([medium.com][1])
* Winston full guide (Better Stack). ([betterstack.com][2])
* GitHub – winston exception/rejection handlers. ([github.com][12])
* Node flag `--unhandled-rejections=strict` discussion. ([answers.netlify.com][7])
* BullMQ docs – failure lifecycle. ([docs.bullmq.io][4])
* axios-retry npm docs. ([npmjs.com][5])
* DEV.to – axios back-off guide. ([dev.to][6])
* SO – JSON logging with timestamp via Winston. ([stackoverflow.com][8])
* SO – request/trace ID logging. ([stackoverflow.com][10])
* Reddit – BullMQ error debugging tips. ([reddit.com][13])
* Node GitHub issue – strict unhandled rejection mode. ([github.com][11])
* StackOverflow – unhandled promise rejection examples. ([stackoverflow.com][9])
* Axios retry back-off alt article (for code refs). ([dev.to][6])
* Winston bug thread on rejection handling nuance. ([github.com][12])

---

### 🧠 Remember

**You own the full error lifecycle:** throw with context → log in JSON → return a clear REST response → retry transients → measure everything.

[1]: https://medium.com/%40xiaominghu19922/proper-error-handling-in-express-server-with-typescript-8cd4ffb67188?utm_source=chatgpt.com "Express Error Handling Like a Pro using Typescript | by Mason Hu"
[2]: https://betterstack.com/community/guides/logging/how-to-install-setup-and-use-winston-and-morgan-to-log-node-js-applications/?utm_source=chatgpt.com "A Complete Guide to Winston Logging in Node.js - Better Stack"
[3]: https://expressjs.com/en/guide/error-handling.html?utm_source=chatgpt.com "Error handling - Express.js"
[4]: https://docs.bullmq.io/guide/workers?utm_source=chatgpt.com "Workers - BullMQ"
[5]: https://www.npmjs.com/package/axios-retry?utm_source=chatgpt.com "axios-retry - NPM"
[6]: https://dev.to/scrapfly_dev/how-to-retry-in-axios-5e87?utm_source=chatgpt.com "How to Retry in Axios - DEV Community"
[7]: https://answers.netlify.com/t/how-to-add-unhandled-rejections-strict/34021?utm_source=chatgpt.com "How to add --unhandled-rejections=strict - Netlify Support Forums"
[8]: https://stackoverflow.com/questions/51975175/log-json-with-date-and-time-format-with-winston-node-logger?utm_source=chatgpt.com "log json with date and time format with winston node logger"
[9]: https://stackoverflow.com/questions/78668429/im-getting-and-unhandled-promise-rejection-that-i-need-correct-syntax-for-in-no?utm_source=chatgpt.com "I'm getting and unhandled promise rejection that I need correct ..."
[10]: https://stackoverflow.com/questions/31087255/how-to-log-request-transaction-id-in-each-log-line-using-winston-for-node-js?utm_source=chatgpt.com "node.js - How to log request & transaction id in each log line using ..."
[11]: https://github.com/nodejs/node/issues/41184?utm_source=chatgpt.com "unexpected behavior with --unhandled-rejections=strict flag #41184"
[12]: https://github.com/winstonjs/winston/issues/2270?utm_source=chatgpt.com "[Bug]: exceptionHandlers and rejectionHandlers logging self and the ..."
[13]: https://www.reddit.com/r/node/comments/1byeyzc/how_to_improve_error_handling_in_bullmq/?utm_source=chatgpt.com "how to improve error handling in bullmq : r/node - Reddit"
