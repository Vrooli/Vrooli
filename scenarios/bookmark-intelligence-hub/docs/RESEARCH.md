# Research

## 2026-06-14 AI Chat Share-Link Extraction (spike, not yet built)

- Reverse-engineered how to pull full conversation content from an AI chat share
  link (ChatGPT). Naive fetch/readability returns blank; the backend API is
  Cloudflare-403'd. The transcript is embedded in the raw HTML as a React Router
  "turbo-stream" payload and must be decoded by index references.
- Captured the working technique + reference decoder + caveats in
  [`CONVERSATION_EXTRACTION.md`](CONVERSATION_EXTRACTION.md) so a future
  "conversation source" integration doesn't reinvent it.

## 2026-05-31 Standards Polish

- `scenario-completeness-scoring score validation bookmark-intelligence-hub` reported two validation-quality issues: one requirement per operational target and a high manual-validation ratio.
- `scenario-auditor standards violations` surfaced cached standards findings for this scenario, including environment defaults in `ui/server.js`, missing API security headers, missing requirement/BAS documentation, and broader UI stack modernization gaps.
- `scenario-auditor standards scan bookmark-intelligence-hub --wait` completed as job `standards-41d34085-f287-4828-96c1-1612a4472c51` and wrote `logs/scenario-auditor/standards/bookmark-intelligence-hub/20260531-213908_job-standards-41d34085-f287-4828-96c1-1612a4472c51.json`.
- `scenario-auditor security scan bookmark-intelligence-hub --wait` completed as job `security-d9a4b48d-c8da-4dce-97e6-dec00d0db303` with 0 vulnerabilities.
