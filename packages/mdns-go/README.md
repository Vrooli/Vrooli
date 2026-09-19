# mdns-go

`mdns-go` provides dependency-free DNS-SD browsing and service advertisement.
The browser sends multicast `PTR` queries, resolves returned instances through
`SRV`, `TXT`, `A`, and `AAAA` records, and returns merged service-instance
observations with their host, addresses, port, and TXT keys. The responder
answers `PTR`, `SRV`, `TXT`, and address queries for a caller-supplied service
without knowing any device, vendor, or service-specific semantics.

Callers provide the service type to `Browse` or `Responder`, and may select the
browse window and interfaces via `Options`. Discovery remains best-effort:
callers must retain their manual endpoint fallback when multicast is blocked.
