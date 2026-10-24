# Authorization adversarial matrix

These are acceptance cases for the verified operator boundary. Each case must
observe both the refusal and the absence of a writer call.

| Attack / failure | Expected result | Evidence |
|---|---|---|
| Missing bearer/cookie | Read-only works; typed mutation returns 401 | authenticator middleware + policygate interceptor tests |
| Expired JWT | 401 before mutation | JWT verifier contract test |
| Wrong issuer or audience | 401 before mutation | JWT verifier contract test |
| `alg=none`/HS256 or unknown `kid` | 401 before mutation | JWT verifier contract test |
| Revoked live session | 401 before mutation | injected Validate/revocation test; live Validate when configured |
| `X-Vrooli-Caller: human` without a token | remains unauthenticated | forged-header boundary test |
| Agent token attempts human intent | 403; no intent row | intent service and authority handler tests |
| Missing intent ID | 401; no Git writer | commit handler test |
| Wrong repository, operation, revision, or subject digest | 409; no Git writer | intent store exact-binding tests |
| Expired or replayed intent | 409; at most one writer call | atomic consume/concurrency tests |
| Restart after issuance | intent remains safe and single-use | SQLite persistence test; raw token ID absent from storage |
| Stale UI/CLI preview | confirmation is re-reviewed or server rejects | intent endpoint recomputation and UI dialog |
| Raw token in logs/errors | absent; only safe principal/reason metadata | logger contract review and middleware design |
