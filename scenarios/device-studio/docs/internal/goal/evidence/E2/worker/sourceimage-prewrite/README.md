# Shared image metadata extraction baseline

Before moving the P2 metadata scanner into `api/internal/sourceimage`, this exact source file was copied here:

- source: `api/internal/projects/image_metadata.go`
- source SHA-256: `382ac0e94dca83ff4a840a2f833b06dd39ed45a93cf2ebb61e94a7ee40ad95bf`
- retained byte copy: `image_metadata.go` (same SHA-256)
- `api/internal/sourceimage/` was absent before the extraction; see `absence.txt`.

The project importer and private worker now share one pure metadata interpretation. No image bytes are transformed by this package.

Final cleanup removed the temporary `projects/image_metadata.go` compatibility wrappers. Project intake now consumes `sourceimage.Inspect` and `sourceimage.ErrUnsupportedImageMetadata` directly. Three parser-only tests moved into `internal/sourceimage`; the two media-intake admission tests remain in `internal/projects`, including rejection of valid nonidentity EXIF orientation. The v6 worker output and receipt were not rewritten by this cleanup.
