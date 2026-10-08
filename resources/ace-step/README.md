# ACE-Step 1.5 resource

This managed service owns one headless ACE-Step 1.5 generation. It uses the
PyTorch backend and keeps the authored caption intact by default. Batching, take
provenance, storage, and pool reservations are owned by `music-tools`.

The resource is Linux amd64/CUDA qualified only. Model and source artifacts are
commit-pinned and checksum-verified by `resource.json`; no model is downloaded
at first request.

