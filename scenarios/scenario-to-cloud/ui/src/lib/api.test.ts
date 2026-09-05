import { expect, describe, test, vi } from "vitest";

import * as api from "./api";

function jsonResponse(payload: unknown, status = 200): Response {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

const opaqueRequest = {} as never;

const successfulCalls: Array<[string, () => Promise<unknown>]> = [
  ["fetchHealth", () => api.fetchHealth()],
  ["validateManifest", () => api.validateManifest({ scenario: { id: "demo" } })],
  ["initManifest", () => api.initManifest()],
  ["buildBundle", () => api.buildBundle({ scenario: { id: "demo" } })],
  ["listBundles", () => api.listBundles()],
  ["getBundleStats", () => api.getBundleStats()],
  ["cleanupBundles", () => api.cleanupBundles(opaqueRequest)],
  ["deleteBundle", () => api.deleteBundle("sha/with spaces")],
  ["listVPSBundles", () => api.listVPSBundles(opaqueRequest)],
  ["deleteVPSBundle", () => api.deleteVPSBundle(opaqueRequest)],
  ["listScenarios", () => api.listScenarios()],
  ["getScenarioPorts", () => api.getScenarioPorts("scenario/with spaces")],
  ["getScenarioDependencies", () => api.getScenarioDependencies("demo")],
  ["checkReachability", () => api.checkReachability("host", "example.com")],
  ["runPreflight", () => api.runPreflight({ scenario: { id: "demo" } })],
  ["fetchSecretsManifest", () => api.fetchSecretsManifest("demo", "desktop", ["postgres", "redis"])],
  ["listDeployments", () => api.listDeployments()],
  ["findInProgressDeployment", () => api.findInProgressDeployment("demo")],
  ["createDeployment", () => api.createDeployment({ scenario: { id: "demo" } }, { name: "demo" })],
  ["getDeployment", () => api.getDeployment("deployment/1")],
  ["executeDeployment", () => api.executeDeployment("deployment/1", { runPreflight: true, forceBundleBuild: false })],
  ["inspectDeployment", () => api.inspectDeployment("deployment/1")],
  ["stopDeployment", () => api.stopDeployment("deployment/1")],
  ["startDeployment", () => api.startDeployment("deployment/1")],
  ["deleteDeployment", () => api.deleteDeployment("deployment/1", { stopOnVPS: true, cleanupBundles: true })],
  ["listSSHKeys", () => api.listSSHKeys()],
  ["generateSSHKey", () => api.generateSSHKey(opaqueRequest)],
  ["getPublicKey", () => api.getPublicKey("/tmp/id_ed25519")],
  ["testSSHConnection", () => api.testSSHConnection(opaqueRequest)],
  ["copySSHKey", () => api.copySSHKey(opaqueRequest)],
  ["deleteSSHKey", () => api.deleteSSHKey(opaqueRequest)],
  ["stopPortServices", () => api.stopPortServices(opaqueRequest)],
  ["openFirewallPorts", () => api.openFirewallPorts(opaqueRequest)],
  ["getDiskUsage", () => api.getDiskUsage(opaqueRequest)],
  ["runDiskCleanup", () => api.runDiskCleanup(opaqueRequest)],
  ["stopScenarioProcesses", () => api.stopScenarioProcesses(opaqueRequest)],
  ["getLiveState", () => api.getLiveState("deployment/1")],
  ["getFiles", () => api.getFiles("deployment/1", "/srv/vrooli")],
  ["getFileContent", () => api.getFileContent("deployment/1", "/srv/vrooli/service.json")],
  ["getDrift", () => api.getDrift("deployment/1")],
  ["killProcess", () => api.killProcess("deployment/1", opaqueRequest)],
  ["restartProcess", () => api.restartProcess("deployment/1", opaqueRequest)],
  ["controlProcess", () => api.controlProcess("deployment/1", opaqueRequest)],
  ["executeVPSAction", () => api.executeVPSAction("deployment/1", opaqueRequest)],
  ["getHistory", () => api.getHistory("deployment/1")],
  ["getLogs", () => api.getLogs("deployment/1", { source: "api", level: "error", tail: 25, search: "failure" })],
  ["checkDNS", () => api.checkDNS("deployment/1")],
  ["getDNSRecords", () => api.getDNSRecords("deployment/1")],
  ["controlCaddy", () => api.controlCaddy("deployment/1", "reload")],
  ["getTLSInfo", () => api.getTLSInfo("deployment/1")],
  ["renewTLS", () => api.renewTLS("deployment/1")],
  ["triggerInvestigation", () => api.triggerInvestigation("deployment/1", opaqueRequest)],
  ["listInvestigations", () => api.listInvestigations("deployment/1", 10)],
  ["getInvestigation", () => api.getInvestigation("deployment/1", "investigation/1")],
  ["stopInvestigation", () => api.stopInvestigation("deployment/1", "investigation/1")],
  ["applyFixes", () => api.applyFixes("deployment/1", "investigation/1", opaqueRequest)],
  ["getAgentManagerStatus", () => api.getAgentManagerStatus()],
  ["createTask", () => api.createTask("deployment/1", opaqueRequest)],
  ["listTasks", () => api.listTasks("deployment/1", 10)],
  ["getTask", () => api.getTask("deployment/1", "task/1")],
  ["stopTask", () => api.stopTask("deployment/1", "task/1")],
  ["listVPSSecrets", () => api.listVPSSecrets("deployment/1")],
  ["getVPSSecret", () => api.getVPSSecret("deployment/1", "API_KEY", true)],
  ["createVPSSecret", () => api.createVPSSecret("deployment/1", "API_KEY", "value", true)],
  ["updateVPSSecret", () => api.updateVPSSecret("deployment/1", "API_KEY", "value", true)],
  ["deleteVPSSecret", () => api.deleteVPSSecret("deployment/1", "API_KEY", true)],
  ["getExpectedSecrets", () => api.getExpectedSecrets("deployment/1", "desktop")],
];

describe("API transport contracts", () => {
  test.each(successfulCalls)("%s returns the backend response on success", async (_name, call) => {
    const fetchMock = vi.fn(async () => jsonResponse({ deployments: [] }));
    vi.stubGlobal("fetch", fetchMock);

    if (_name === "findInProgressDeployment") {
      await expect(call()).resolves.toBeNull();
    } else {
      await expect(call()).resolves.toEqual({ deployments: [] });
    }
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  test("every operation fails closed when the backend rejects the request", async () => {
    const fetchMock = vi.fn(async () => jsonResponse({ error: "unavailable" }, 503));
    vi.stubGlobal("fetch", fetchMock);

    for (const [name, call] of successfulCalls) {
      await expect(call(), name).rejects.toThrow();
    }

    expect(fetchMock).toHaveBeenCalledTimes(successfulCalls.length);
  });
});
