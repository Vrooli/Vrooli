# mdns-go

`mdns-go` is a read-only DNS-SD browser. It sends multicast `PTR` queries,
resolves returned instances through `SRV`, `TXT`, `A`, and `AAAA` records, and
returns merged service-instance observations with their host, addresses, port,
and TXT keys.

The package browses; it does not advertise services, publish records, answer
queries, or contain knowledge of any device, vendor, or service type. Callers
provide the service types to `Browse` and may select the browse window and
interfaces via `Options`.
