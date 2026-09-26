# Error Handling

## Proto-Typed Operations

Generated Connect surfaces should return typed graph responses with per-source errors instead of hiding partial upstream failures.

## Sentinel Mapping

Domain errors should map to stable HTTP/Connect status codes and user-facing messages.

## Multipart REST Exceptions

SDA does not currently expose multipart upload workflows.

## Cross-References

- `SEAMS.md`
- `../reference/api-endpoints.md`
