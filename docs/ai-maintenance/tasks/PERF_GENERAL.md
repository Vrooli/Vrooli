## 🚀 Quick Take

General performance hinges on four pillars—**event-loop discipline, efficient data access, memory hygiene, and algorithmic thrift**. By profiling hot paths, fixing event-loop blockers, tightening database/index usage, and rooting out leaks, you can cut p95 latencies and keep CPU + RAM headroom high. The practices here draw on the Node.js core docs, V8 internals, modern Postgres/Redis tuning guides, and real-world production war stories. ([github.com][1], [nodejs.org][2], [mydbops.com][3], [learn.redis.com][4], [medium.com][5], [medium.com][6])

---

## 1 Purpose & Scope

* **Task ID:** `PERF_GENERAL`
* **Goal:** Find and fix performance smells **outside** React rendering (that’s `REACT_PERF`).
* **Includes:** Node CPU/IO bottlenecks, DB latency, Redis round-trips, memory leaks, algorithmic complexity, logging overhead.
* **Excludes:** Front-end FPS issues, CSS reflow, React memoisation (see `REACT_PERF`).

---

## 2 Hard Rules

| Rule                                                                     | Why |
| ------------------------------------------------------------------------ | --- |
| Don’t change observable behaviour—functional parity is mandatory.        |     |
| No Git commands; leave diffs uncommitted for human review.               |     |
| Avoid new deps unless they demonstrably replace home-grown slow code.    |     |
| Keep `.js` extensions in TS imports (Node ESM).                          |     |
| Always back performance claims with **benchmarks or profiler evidence**. |     |

---

## 3 Performance Pillars & Quick Wins

### 3.1 Event Loop Discipline

* **Avoid sync CPU on main thread.** Look for rogue JSON parsing, large regexes, and crypto on the event loop. ([nodejs.org][2], [noncodersuccess.medium.com][7])
* Off-load heavy work to **Worker Threads** or a job queue. ([aditya-sunjava.medium.com][8])
* Use **non-blocking I/O** everywhere; never wrap fs/promises calls in sync fallbacks. ([github.com][1])

### 3.2 Efficient Data Access

* **Index ruthlessly.** Add b-tree, GIN, or partial indexes to match WHERE clauses; drop unused ones to keep writes fast. ([mydbops.com][3], [medium.com][9])
* Batch Redis reads/writes with **pipelining or Lua scripts**; cut RTTs. ([learn.redis.com][4])
* Right-size **connection pools**—too small starves, too big thrashes. Use pool-level metrics to tune. ([medium.com][10])

### 3.3 Memory Hygiene

* Regularly capture **heap snapshots** (Chrome DevTools `chrome://inspect`) to spot detached DOM-like objects and listeners. ([medium.com][5], [sematext.com][11])
* Use `--max-old-space-size` only after you’ve fixed leaks, not to hide them. ([sematext.com][11])
* Prefer object shapes that keep **V8 hidden classes stable** (consistent property order, avoid polymorphic objects). ([v8.dev][12], [v8.dev][13])

### 3.4 Algorithmic Thrift

* Audit O(n²) loops; refactor to maps or sets. Brush up on Big-O and use built-in methods (`Map.has`, `Set`) for O(1) lookups. ([medium.com][6])
* Cache expensive pure computations (memoisation or Redis) when hit ratio is high. ([aditya-sunjava.medium.com][8])
* Throttle/-debounce logging—excessive structured logs can add 10-30 ms per request. ([aditya-sunjava.medium.com][8], [github.com][1])

---

## 4 Detection & Tooling

| Task            | Tool / Command                                                | Notes                                               |
| --------------- | ------------------------------------------------------------- | --------------------------------------------------- |
| CPU hotspots    | `node --inspect` + DevTools **Flame Chart** or `clinic flame` | Locate ≥5 ms JS frames. ([nodejs.org][2])           |
| I/O latency     | `autocannon -c 100` before/after change                       | Include p95 & p99. ([aditya-sunjava.medium.com][8]) |
| DB slow queries | `EXPLAIN ANALYZE SELECT …` + `pg_stat_statements`             | Look for seq scans > 1 k row. ([mydbops.com][3])    |
| Redis RTT       | `redis-cli --latency -i 1`                                    | Aim P50 < 1 ms inside VPC. ([learn.redis.com][4])   |
| Memory leaks    | `node --inspect --expose-gc` → Heap Snapshot diff             | Growth > 10 MB/min is suspect. ([medium.com][5])    |

---

## 5 Step-by-Step Workflow

1. **Profile first.** Never optimise blind.

2. **Pick the top offender** (≥20 % of CPU time or >p95 latency).

3. Apply the relevant quick win from §3.

4. **Bench again** (`autocannon`, unit perf tests) and record delta.

5. Update or insert:

   ```ts
   // AI_CHECK: PERF_GENERAL=<incrementedCount> | LAST: 2025-06-16
   ```

6. Write one-line stdout summary: `queueFactory.ts → cut Redis round-trips 4→1 (p95 -12 ms).`

---

## 6 Query Cookbook

```bash
# Files never reviewed for performance
rg -L "AI_CHECK:.*PERF_GENERAL" packages -t ts -t tsx

# Hot modules by import count (proxy for critical path)
rg --no-heading -o 'from .+["\'](@vrooli[^"\']+)["\']' packages \
  | sort | uniq -c | sort -nr | head -20
```

---

## 7 Fast Checklist

| ✔︎                                             | Item |
| ---------------------------------------------- | ---- |
| You have profiler or benchmark evidence.       |      |
| Change does not alter behaviour or public API. |      |
| Tests + lints green.                           |      |
| `AI_CHECK` tag updated for `PERF_GENERAL`.     |      |

---

## 8 Reference Links

1. Node.js Best Practices list (performance section) ([github.com][1])
2. Node.js docs—don’t block the event loop ([nodejs.org][2])
3. Medium—blocking event loop pitfalls & fixes ([noncodersuccess.medium.com][7])
4. PostgreSQL indexing guide (2025) ([mydbops.com][3])
5. SQL/Postgres best practices (2025) ([medium.com][9])
6. Redis performance tuning (pipelining) ([learn.redis.com][4])
7. Connection-pool best practices ([medium.com][10])
8. Memory-leak detection tutorial ([medium.com][5])
9. Sematext Node memory-leak guide ([sematext.com][11])
10. V8 hidden-class docs ([v8.dev][12])
11. V8 faster class-features post ([v8.dev][13])
12. Big-O in JS article ([medium.com][6])
13. Node perf best practices (async, caching) ([aditya-sunjava.medium.com][8])

---

### Remember

Profile → Fix → Re-measure → Document. Small, measured wins compound into big p99 gains over time.

[1]: https://github.com/goldbergyoni/nodebestpractices?utm_source=chatgpt.com "white_check_mark: The Node.js best practices list (July 2024) - GitHub"
[2]: https://nodejs.org/en/learn/asynchronous-work/dont-block-the-event-loop?utm_source=chatgpt.com "Don't Block the Event Loop (or the Worker Pool) - Node.js"
[3]: https://www.mydbops.com/blog/postgresql-indexing-best-practices-guide?utm_source=chatgpt.com "PostgreSQL Index Best Practices for Faster Queries - Mydbops"
[4]: https://learn.redis.com/kb/doc/1mebipyp1e/performance-tuning-best-practices?utm_source=chatgpt.com "Performance Tuning Best Practices - Redis"
[5]: https://medium.com/%40amirilovic/how-to-find-production-memory-leaks-in-node-js-applications-a1b363b4884f?utm_source=chatgpt.com "How to find production memory leaks in Node.js applications?"
[6]: https://medium.com/zorak-laptop/understanding-big-o-notation-in-javascript-a-comprehensive-guide-for-developers-faac7a65b7d7?utm_source=chatgpt.com "Understanding Big O Notation in JavaScript - Medium"
[7]: https://noncodersuccess.medium.com/blocking-the-event-loop-in-node-js-understanding-and-avoiding-performance-pitfalls-95fd287a1397?utm_source=chatgpt.com "Blocking the Event Loop in Node.js: Understanding and Avoiding ..."
[8]: https://aditya-sunjava.medium.com/optimizing-node-js-performance-best-practices-for-speed-and-scalability-13853a341799?utm_source=chatgpt.com "Optimizing Node.js Performance: Best Practices for Speed and ..."
[9]: https://medium.com/learning-data/best-practices-for-sql-postgresql-in-2025-keep-your-database-from-crying-317eaf06f688?utm_source=chatgpt.com "Best Practices for SQL & PostgreSQL in 2025: Keep Your Database ..."
[10]: https://medium.com/%40aggelosbellos/database-connection-pooling-optimizing-database-interactions-for-performance-and-scalability-62d95a1f7b4c?utm_source=chatgpt.com "Database Connection Pooling: Optimizing Database Interactions for ..."
[11]: https://sematext.com/blog/nodejs-memory-leaks/?utm_source=chatgpt.com "Node.js Memory Leak Detection: How to Debug & Avoid Them"
[12]: https://v8.dev/docs/hidden-classes?utm_source=chatgpt.com "Maps (Hidden Classes) in V8 - V8.dev"
[13]: https://v8.dev/blog/faster-class-features?utm_source=chatgpt.com "Faster initialization of instances with new class features - V8.dev"
