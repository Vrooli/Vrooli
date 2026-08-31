# macOS speech artifact qualification

The local `whisper` and `kokoro` resources remain unsupported on macOS until
each native artifact has all of this evidence:

1. A native server executable for the target architecture.
2. A signed macOS build that launches on a clean host.
3. A pinned SHA-256 checksum for the exact artifact.
4. A recorded smoke run proving the health endpoint and speech operation.

The audio-tools capability qualification harness reports each unmet item. A
BYOK speech provider is the supported macOS path while these local artifacts
remain unqualified.
