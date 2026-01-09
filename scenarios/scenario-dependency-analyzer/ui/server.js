#!/usr/bin/env node
import path from "node:path";
import { fileURLToPath } from "node:url";
import { createScenarioServer } from "@vrooli/api-base/server";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const uiPort = process.env.UI_PORT ?? "36897";
const apiPort = process.env.API_PORT ?? "15533";

const app = createScenarioServer({
  uiPort,
  apiPort,
  distDir: path.join(__dirname, "dist"),
  serviceName: "scenario-dependency-analyzer",
  version: process.env.npm_package_version ?? "1.0.0",
  corsOrigins: "*"
});

const normalizedPort = Number.parseInt(uiPort, 10);
if (Number.isNaN(normalizedPort)) {
  throw new Error(`Invalid UI_PORT: ${uiPort}`);
}

app.listen(normalizedPort, "0.0.0.0", () => {
  console.log(`Scenario Dependency Analyzer UI listening on http://localhost:${normalizedPort}`);
  console.log(`Proxying API requests to http://localhost:${apiPort}`);
});
