import { createScenarioConnectTransport, resolveApiBase } from "@vrooli/api-base";

// Resolve the scenario base exactly once so Connect and the small ops-probe
// exception share identical proxy/tunnel behavior.
export const scenarioApiBase = resolveApiBase({ appendSuffix: true });
export const scenarioTransport = createScenarioConnectTransport({ baseUrl: scenarioApiBase });
