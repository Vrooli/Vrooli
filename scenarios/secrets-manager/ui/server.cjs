#!/usr/bin/env node
/**
 * Lifecycle-managed production server for the Secrets Manager UI.
 *
 * The shared server owns health, static files, SPA fallback, and /api proxying
 * so localhost, tunnel, and embedded-proxy requests follow the same path.
 */
const path = require("node:path");

async function start() {
  const { startScenarioServer } = await import("@vrooli/api-base/server");

  startScenarioServer({
    uiPort: process.env.UI_PORT,
    apiPort: process.env.API_PORT,
    distDir: path.join(__dirname, "dist"),
    serviceName: "secrets-manager",
    corsOrigins: "*"
  });
}

start().catch((error) => {
  console.error("Failed to start Secrets Manager UI server:", error);
  process.exit(1);
});
