# Error Handling

## Request Validation

Handlers return `400` when required parameters or JSON bodies are invalid. Examples include missing `region_id` for foliage, weather, and reports endpoints.

## Method Validation

Unsupported methods return `405` for prediction, reports, and trips handlers.

## Dependency Degradation

Region and foliage reads can fall back to sample data when PostgreSQL is unavailable. Prediction requests can fall back to typical peak weeks when Ollama is unavailable or returns invalid output.

## Fatal Conditions

Write paths for reports and trips require a database connection. They should report an error rather than silently pretending to persist user content.
