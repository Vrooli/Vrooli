# API Endpoints

- `GET /api/v2/scenarios` lists manifest-derived scenario choices.
- `GET /api/v2/host-requirements` returns host tools and safeguards.
- `GET /api/v2/readiness` returns actionable metadata-safe readiness.
- `POST /api/v2/credentials/provision` provisions declared credentials.
- `GET|PUT /api/v1/operator-state` reads or atomically commits operator choices.

No endpoint returns a credential value.
