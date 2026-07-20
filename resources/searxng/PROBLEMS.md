# Known operational limits

- External search engines can rate-limit, CAPTCHA, or time out. This is an
  upstream availability condition, not a resource lifecycle failure.
  `resource-searxng engine-health --json` reports degraded engines while the
  `web-search` consumer continues to handle unavailable live search gracefully.
- Docker is required by design. Hosts without a running Docker daemon are not
  supported for this resource and should fail the control-plane preflight
  before attempting runtime operations.
