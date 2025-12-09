# Kids Mode Dashboard

## Overview
A simple web dashboard for kids, providing a safe and engaging interface for educational content.

## Setup

1. Install dependencies:
   ```
   ./cli/install.sh
   ```

2. Build the API:
   ```
   cd api
   go build -o kids-mode-dashboard-api .
   ```

3. Run tests:
   ```
   vrooli scenario test kids-mode-dashboard
   ```

## Usage

- Start the server:
  ```
  cd api
  ./kids-mode-dashboard-api
  ```

- Access the dashboard at: http://localhost:8080

- Check service health at: http://localhost:8080/api/v1/health

## Development

- Edit `api/main.go` for backend changes
- The frontend is embedded in the Go binary for simplicity

## Testing

Run `vrooli scenario test kids-mode-dashboard` (or `make test`) to execute all test phases.
