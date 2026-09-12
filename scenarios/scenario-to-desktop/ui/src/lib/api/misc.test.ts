import { describe, expect, it, vi } from "vitest";

const clients = vi.hoisted(() => ({
  getDocumentationManifest: vi.fn(),
  getDocumentationContent: vi.fn(),
  listDesktopRecords: vi.fn(),
  moveDesktopRecord: vi.fn(),
  deleteDesktopScenario: vi.fn(),
  probeEndpoints: vi.fn(),
  getProxyHints: vi.fn(),
  resolveScenarioPort: vi.fn(),
  inspectManifest: vi.fn(),
  checkWine: vi.fn(),
  installWine: vi.fn(),
  getWineInstallStatus: vi.fn(),
  getTelemetryInsights: vi.fn(),
  ingestTelemetry: vi.fn(),
  deleteTelemetry: vi.fn(),
  getTelemetrySummary: vi.fn(),
  getTelemetryTail: vi.fn(),
}));

vi.mock("./connect", () => ({
  documentationConnectClient: clients,
  desktopRecordsConnectClient: clients,
  operationsConnectClient: clients,
  preflightConnectClient: clients,
  systemConnectClient: clients,
  telemetryConnectClient: clients,
}));
vi.mock("./client", () => ({
  buildUrl: (path: string) => `https://desktop.test${path}`,
}));

import {
  checkWineStatus,
  deleteDesktopBuild,
  deleteTelemetry,
  fetchBundleManifest,
  fetchDocContent,
  fetchDocsManifest,
  fetchProxyHints,
  fetchScenarioPort,
  fetchTelemetryInsights,
  fetchTelemetrySummary,
  fetchTelemetryTail,
  fetchWineInstallStatus,
  getDownloadUrl,
  getIconPreviewUrl,
  getTelemetryDownloadUrl,
  moveDesktopRecord,
  probeEndpoints,
  startWineInstall,
  uploadTelemetry,
} from "./misc";

describe("miscellaneous generated API clients", () => {
  it("routes documentation, records, probes, manifests, and runtime helpers through Connect", async () => {
    clients.getDocumentationManifest.mockResolvedValue({ sections: [] });
    clients.getDocumentationContent.mockResolvedValue({ content: "# Docs" });
    clients.moveDesktopRecord.mockResolvedValue({
      recordId: "record-1",
      from: "/old",
      to: "/new",
      status: "moved",
    });
    clients.deleteDesktopScenario.mockResolvedValue({ status: "deleted" });
    clients.probeEndpoints.mockResolvedValue({ reachable: true });
    clients.getProxyHints.mockResolvedValue({ hints: [] });
    clients.resolveScenarioPort.mockResolvedValue({ port: 8080 });
    clients.inspectManifest
      .mockResolvedValueOnce({ errors: [], manifest: { name: "Calculator" } })
      .mockResolvedValueOnce({ errors: [{ message: "manifest is invalid" }] });
    clients.checkWine.mockResolvedValue({ installed: true });
    clients.installWine.mockResolvedValue({ installId: "wine-1" });
    clients.getWineInstallStatus.mockResolvedValue({ status: "complete" });

    await expect(fetchDocsManifest()).resolves.toEqual({ sections: [] });
    await expect(fetchDocContent("operator.md")).resolves.toEqual({
      content: "# Docs",
    });
    await expect(
      moveDesktopRecord("record-1", {
        target: "custom",
        destination_path: "/new",
      }),
    ).resolves.toMatchObject({ record_id: "record-1", to: "/new" });
    await expect(deleteDesktopBuild("calculator")).resolves.toEqual({
      status: "deleted",
    });
    await expect(
      probeEndpoints({ proxy_url: "http://localhost:8080", timeout_ms: 500 }),
    ).resolves.toEqual({ reachable: true });
    await expect(fetchProxyHints("calculator")).resolves.toEqual({ hints: [] });
    await expect(fetchScenarioPort("calculator", "api")).resolves.toEqual({
      port: 8080,
    });
    await expect(
      fetchBundleManifest({ bundle_manifest_path: "/tmp/bundle.json" }),
    ).resolves.toMatchObject({ manifest: { name: "Calculator" } });
    await expect(
      fetchBundleManifest({ bundle_manifest_path: "/tmp/broken.json" }),
    ).rejects.toThrow("manifest is invalid");
    await expect(checkWineStatus()).resolves.toEqual({ installed: true });
    await expect(startWineInstall("apt")).resolves.toEqual({
      install_id: "wine-1",
    });
    await expect(fetchWineInstallStatus("wine-1")).resolves.toEqual({
      status: "complete",
    });

    expect(clients.moveDesktopRecord).toHaveBeenCalledWith({
      recordId: "record-1",
      target: "custom",
      destinationPath: "/new",
    });
    expect(clients.probeEndpoints).toHaveBeenCalledWith({
      proxyUrl: "http://localhost:8080",
      serverUrl: undefined,
      apiUrl: undefined,
      timeoutMs: 500,
    });
    expect(clients.inspectManifest).toHaveBeenCalledWith({
      manifestPath: "/tmp/bundle.json",
    });
  });

  it("validates telemetry payloads and preserves the declared download URL contract", async () => {
    clients.getTelemetryInsights.mockResolvedValue({
      payload: { scenario_name: "calculator", exists: true },
    });
    clients.ingestTelemetry.mockResolvedValue({
      outputPath: "/tmp/events.jsonl",
    });
    clients.getTelemetrySummary.mockResolvedValue({
      payload: { scenario_name: "calculator", exists: true, event_count: 1 },
    });
    clients.getTelemetryTail.mockResolvedValue({
      payload: {
        scenario_name: "calculator",
        exists: true,
        limit: 5,
        entries: [],
      },
    });

    await expect(fetchTelemetryInsights("calculator")).resolves.toMatchObject({
      exists: true,
    });
    await expect(
      uploadTelemetry({
        scenario_name: "calculator",
        events: [{ event: "app_ready", details: { port: 8080 } }],
      }),
    ).resolves.toEqual({ output_path: "/tmp/events.jsonl" });
    await deleteTelemetry("calculator");
    await expect(fetchTelemetrySummary("calculator")).resolves.toMatchObject({
      event_count: 1,
    });
    await expect(fetchTelemetryTail("calculator", 5)).resolves.toMatchObject({
      limit: 5,
    });

    expect(clients.ingestTelemetry).toHaveBeenCalledWith({
      scenarioName: "calculator",
      deploymentMode: "external-server",
      source: "desktop-upload",
      events: [{ event: "app_ready", details: { port: 8080 } }],
    });
    expect(clients.deleteTelemetry).toHaveBeenCalledWith({
      scenarioName: "calculator",
    });
    expect(getIconPreviewUrl("icons/My Icon.svg")).toBe(
      "https://desktop.test/icons/preview?path=icons%2FMy%20Icon.svg",
    );
    expect(getDownloadUrl("calculator", "linux")).toBe(
      "https://desktop.test/desktop/download/calculator/linux",
    );
    expect(getTelemetryDownloadUrl("calculator/one")).toBe(
      "https://desktop.test/deployment/telemetry/calculator%2Fone/download",
    );
  });

  it("fails closed when telemetry Connect responses omit their required structured payload", async () => {
    clients.getTelemetrySummary.mockResolvedValue({});
    clients.getTelemetryTail.mockResolvedValue({ payload: undefined });

    await expect(fetchTelemetrySummary("calculator")).rejects.toThrow(
      "Connect service returned an empty payload",
    );
    await expect(fetchTelemetryTail("calculator")).rejects.toThrow(
      "Connect service returned an empty payload",
    );
  });
});
