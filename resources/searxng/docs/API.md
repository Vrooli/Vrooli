# HTTP consumer contract

The stable consumer surface is SearXNG's HTTP API at `SEARXNG_URL` (default
`http://localhost:8280`). `web-search` uses this contract directly.

```text
GET /search?q=<url-encoded-query>&format=json
Accept: application/json
```

Consumers must tolerate network failures, non-2xx responses, empty result
sets, and `unresponsive_engines` entries in the JSON envelope. The resource's
`engine-health` command uses the same JSON format for an availability smoke;
external search engines remain nondeterministic and are not asserted by result
count.
