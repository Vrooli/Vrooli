# Tech Tree Designer Documentation

This scenario presents the Tech Tree Designer experience. It combines a rich React front-end with a Go API and PostgreSQL storage so that agents can explore and restructure the civilization-scale technology roadmap.

## Key Concepts
- **Interactive Graph:** The ReactFlow canvas renders sectors and progression stages with sector-aware styling, selectable stage details, and mini-map navigation.
- **Edit Mode:** Toggle edit mode to reposition stages or wire new dependencies directly on the canvas. Changes persist via the `/api/v1/tech-tree/graph` endpoint.
- **Strategic Context:** The surrounding UI surfaces sector progress, milestone roadmaps, and AI-driven recommendations sourced from the API service.

## Development Commands
- `make start` – launch the scenario through the lifecycle system
- `make test` – execute the phased test suite
- `make logs` – tail lifecycle-managed logs
- `vrooli scenario restart tech-tree-designer` – restart after code changes

## API Highlights
- `GET /api/v1/tech-tree/sectors` – list sectors and stages
- `GET /api/v1/dependencies` – inspect stage dependencies
- `PUT /api/v1/tech-tree/graph` – persist stage positions and dependency updates from the editor

## Front-End Notes
- Located in `ui/`, built with Vite, React 18, and ReactFlow 10.
- Editing state uses `useNodesState`/`useEdgesState` to stay in sync with persisted graph data.

## Back-End Notes
- Go API in `api/`, backed by PostgreSQL schema under `initialization/postgres`.
- Integration tests in `api/integration_test.go` validate graph editing, progress tracking, and strategic analysis routes.

## Next Steps
- Extend edit mode with metadata editing (stage type, descriptions).
- Layer validation to guard against circular or duplicate dependencies during editing.

