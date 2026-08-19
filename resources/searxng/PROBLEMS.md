# Known operational limits

- External search engines can rate-limit, CAPTCHA, or time out. This is an
  upstream availability condition, not a resource lifecycle failure.
  `resource-searxng engine-health --json` reports degraded engines while the
  `web-search` consumer continues to handle unavailable live search gracefully.
- Native runtime smoke coverage on each non-Linux desktop target remains a
  release-pipeline responsibility; the manifest still carries target-specific
  runtime and composed-tree digests so a missing or altered artifact fails
  before launch.
