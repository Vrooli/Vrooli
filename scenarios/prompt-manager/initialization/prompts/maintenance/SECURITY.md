## 🚀 Executive Summary

Modern JavaScript apps face supply-chain attacks, secret leaks, insecure headers, injection, XSS, and denial-of-service. **Your goal is to find and neutralise those risks without changing app behaviour**: scan code and dependencies, harden configuration, add guards, and track progress with `AI_CHECK` comments.

---

## 1 Scope & Hard Rules

* **Task ID:** `SECURITY`
* **Focus:** Codebase, config, CI, and dependency risks that can be fixed in-place.
* **Out-of-scope:** Large infra changes, IAM re-architecture—raise TODOs instead.

| Rule                                             | Why                                 |
| ------------------------------------------------ | ----------------------------------- |
| Never run Git commands                           | Human review required.              |
| Don’t add new deps without approval              | Prevents further supply-chain risk. |
| Keep `.js` extensions in TS imports              | Node ESM.                           |
| Prefer least-privilege, secure-by-default fixes. |                                     |

---

## 2 Threat Categories & Fast Fixes

| Category                                                             | How to Detect                                  | Typical Fix                                                                                       |
| -------------------------------------------------------------------- | ---------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| **Known-vuln packages**                                              | `pnpm audit --fix` or CI Action                | Update / override version. ([pnpm.io][1], [github.com][2])                                        |
| **Secrets in repo**                                                  | `trufflehog filesystem --since-commit HEAD~30` | Remove / rotate secret; add `.gitignore` rule. ([github.com][3], [trufflesecurity.com][4])        |
| **Unsafe code patterns** (eval, regex DoS, SQL injection, XSS sinks) | `semgrep --config=p/security-audit`            | Refactor; use parameterised APIs. ([semgrep.dev][5], [semgrep.dev][6])                            |
| **Missing security headers**                                         | Search for Express `app.use(helmet())`         | Add & configure Helmet. ([habtesoft.medium.com][7], [github.com][8])                              |
| **Rate-limit gaps**                                                  | Check for `express-rate-limit` use             | Add rate-limit & slowdown middleware. ([developer.mozilla.org][9], [reddit.com][10])              |
| **Supply-chain confusion**                                           | Review private package names vs npm            | Reserve names / use npm-scopes. ([snyk.io][11], [websecuritylens.org][12])                        |
| **ESLint security hotspots**                                         | `eslint --ext .ts --plugin security`           | Fix or flag issues. ([npmjs.com][13], [npmjs.com][14])                                            |
| **OWASP Top 10 risks**                                               | Cross-map findings to OWASP cheatsheets        | Ensure controls for auth, an-& authorisation. ([cheatsheetseries.owasp.org][15], [owasp.org][16]) |

---

## 3 Toolbelt Setup

### 3.1 Automated Scanners

1. **pnpm audit CI** – fail PR if high/critical vulns.
2. **Semgrep** – run `p/typescript` + `p/security-audit`.
3. **trufflehog** – secret scan on diff and full repo.

### 3.2 Lint Rules

```jsonc
{
  "plugins": ["security", "@typescript-eslint"],
  "extends": ["plugin:security-node/recommended"],
  "rules": {
    "@typescript-eslint/no-implied-eval": "error",
    "security/detect-object-injection": "error"
  }
}
```

---

## 4 Step-by-Step Workflow

1. **Scan & Collect Findings**

   ```bash
   pnpm audit
   semgrep --config=p/security-audit --json > .semgrep.json
   trufflehog git --max-depth 50
   ```

2. **Prioritise**

   * High/critical CVEs with known fixes.
   * Leaked secrets.
   * Code sinks leading to injection/XSS.

3. **Patch**

   * Upgrade / pin package.
   * Rotate credentials; commit `.env.example`.
   * Refactor code to parameterised queries, escape HTML, remove eval, enforce CSP.

4. **Verify**

   * `pnpm test && pnpm tsc --noEmit && pnpm lint`.
   * Rerun scanners; ensure clean.

5. **Update AI\_CHECK**

   ```ts
   // AI_CHECK: SECURITY=<incrementedCount> | LAST: 2025-06-16
   ```

6. **Write one-line summary to stdout** (`users.service.ts → removed raw SQL, switched to Prisma parameterised query`).

---

## 5 Coding Guidelines

### 5.1 Dependency Hygiene

* Stick to semver ranges that exclude known CVEs; lockfile required.
* Prefer “ES Modules only” libs—fewer polyfill surprises.

### 5.2 Secrets Management

* Load via env vars; never default to plaintext.
* Support 12-factor patterns so CI/CD can inject secrets.

### 5.3 Injection Defence

* All DB access through Prisma’s parameterised APIs (prevents SQLi). ([semgrep.dev][6])
* Escape or sanitise any HTML with DOMPurify when dangerously injecting.

### 5.4 Secure Headers & DoS Mitigation

* `helmet()` with CSP, HSTS, Referrer-Policy, X-Frame-Options.
* `express-rate-limit` (or open-source alternative) on auth and public APIs.

### 5.5 Supply-Chain Controls

* Use **npm scopes** (e.g., `@vrooli/xyz`) for internal packages.
* Mirror critical third-party packages in your own registry if possible.

---

## 6 Query Cookbook

```bash
# Files never reviewed for SECURITY
rg -L "AI_CHECK:.*SECURITY" packages -t ts -t tsx

# Find inline eval or Function()
rg -n "eval\\(|new Function\\(" packages

# Find Express apps missing Helmet
rg -L "helmet(" --glob "packages/*/src/**/*.ts"

# Check for raw SQL strings
rg "\"SELECT .* FROM" packages
```

---

## 7 Fast Checklist

| ✔︎                                               | Item |
| ------------------------------------------------ | ---- |
| No high/critical vulns from `pnpm audit`.        |      |
| No secrets in working tree (`trufflehog` clean). |      |
| Lint & Semgrep pass with zero --severity ERROR.  |      |
| Security headers & rate limits configured.       |      |
| `AI_CHECK` updated (only for SECURITY).          |      |

---

## 8 Reference Cheat-sheet Links

* OWASP Node.js Cheat Sheet ([cheatsheetseries.owasp.org][15])
* OWASP Top 10 (2025 draft) ([owasp.org][16])
* Semgrep TypeScript ruleset ([semgrep.dev][5])
* pnpm audit docs & GitHub Action ([pnpm.io][1], [github.com][2])
* TruffleHog secrets scanning ([github.com][3], [trufflesecurity.com][4])
* ESLint security plugins ([npmjs.com][13], [npmjs.com][14])
* Dependency-confusion overview & prevention ([snyk.io][11], [websecuritylens.org][12])
* Helmet middleware guide & repo ([habtesoft.medium.com][7], [github.com][8])
* Express rate-limit best practices (MDN) ([developer.mozilla.org][9])

---

### Remember

> *Defence in depth wins.* Each small patch—upgrading a package, adding a header, deleting an accidental key—shrinks the attack surface and buys you peace of mind.

[1]: https://pnpm.io/cli/audit?utm_source=chatgpt.com "pnpm audit"
[2]: https://github.com/marketplace/actions/pnpm-audit?utm_source=chatgpt.com "PNPM Audit · Actions · GitHub Marketplace"
[3]: https://github.com/trufflesecurity/trufflehog?utm_source=chatgpt.com "trufflesecurity/trufflehog: Find, verify, and analyze leaked credentials"
[4]: https://trufflesecurity.com/trufflehog?utm_source=chatgpt.com "What is TruffleHog? Truffle Security Co."
[5]: https://semgrep.dev/p/typescript?utm_source=chatgpt.com "typescript ruleset - Semgrep"
[6]: https://semgrep.dev/p/security-audit?utm_source=chatgpt.com "security-audit ruleset - Semgrep"
[7]: https://habtesoft.medium.com/helmet-enhancing-security-in-node-js-14f31763a7dd?utm_source=chatgpt.com "Helmet: Enhancing Security in Node.js | by habtesoft - Medium"
[8]: https://github.com/helmetjs/helmet?utm_source=chatgpt.com "helmetjs/helmet: Help secure Express apps with various ... - GitHub"
[9]: https://developer.mozilla.org/en-US/blog/securing-apis-express-rate-limit-and-slow-down/?utm_source=chatgpt.com "Securing APIs: Express rate limit and slow down | MDN Blog"
[10]: https://www.reddit.com/r/node/comments/182a07g/what_are_the_simplest_strategies_to_implement/?utm_source=chatgpt.com "What are the simplest strategies to implement rate limiting for ... - Reddit"
[11]: https://snyk.io/blog/detect-prevent-dependency-confusion-attacks-npm-supply-chain-security/?utm_source=chatgpt.com "Detect and prevent dependency confusion attacks on npm to ... - Snyk"
[12]: https://www.websecuritylens.org/how-dependency-confusion-attack-works-and-how-to-prevent-it/?utm_source=chatgpt.com "How Dependency Confusion attack works and How to prevent it"
[13]: https://www.npmjs.com/package/eslint-plugin-security-rules?utm_source=chatgpt.com "eslint-plugin-security-rules - NPM"
[14]: https://www.npmjs.com/package/eslint-plugin-security?utm_source=chatgpt.com "eslint-plugin-security - NPM"
[15]: https://cheatsheetseries.owasp.org/cheatsheets/Nodejs_Security_Cheat_Sheet.html?utm_source=chatgpt.com "Nodejs Security - OWASP Cheat Sheet Series"
[16]: https://owasp.org/www-project-top-ten/?utm_source=chatgpt.com "OWASP Top Ten"
