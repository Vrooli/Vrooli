import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import SettingsDialog from "./SettingsDialog";

const settingsMocks = vi.hoisted(() => ({
  fetchConfig: vi.fn(),
  updateConfig: vi.fn(),
  fetchDefaults: vi.fn(),
  exportConfig: vi.fn(),
  importConfig: vi.fn(),
  setCheckEnabled: vi.fn(),
  setCheckAutoHeal: vi.fn(),
  bulkUpdateChecks: vi.fn(),
  fetchChecks: vi.fn(),
  fetchMonitoring: vi.fn(),
  addScenario: vi.fn(),
  removeScenario: vi.fn(),
  setScenarioCritical: vi.fn(),
  addResource: vi.fn(),
  removeResource: vi.fn(),
}));

vi.mock("../../lib/api", () => settingsMocks);

const config = {
  version: "1.0",
  global: {
    gracePeriodSeconds: 60,
    tickIntervalSeconds: 60,
    verifyDelaySeconds: 30,
    maxRestartAttempts: 3,
    restartCooldownSeconds: 300,
    historyRetentionHours: 24,
  },
  checks: { "infra-dns": { enabled: true, autoHeal: true } },
  ui: { autoRefreshSeconds: 30, theme: "system", showDisabledChecks: false, defaultTab: "dashboard" },
};

const defaults = { global: config.global, ui: config.ui, checks: { "infra-dns": { enabled: true, autoHeal: true, intervalSeconds: 30 } } };

describe("SettingsDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    settingsMocks.fetchConfig.mockResolvedValue(config);
    settingsMocks.fetchDefaults.mockResolvedValue(defaults);
    settingsMocks.fetchChecks.mockResolvedValue([
      { id: "infra-dns", title: "DNS", description: "DNS health", importance: "required", category: "infrastructure", intervalSeconds: 30 },
      { id: "resource-postgres", title: "Postgres", description: "Database", importance: "required", category: "resource", intervalSeconds: 60 },
    ]);
    settingsMocks.fetchMonitoring.mockResolvedValue({ scenarios: { app: { critical: true }, demo: { critical: false } }, resources: ["redis", "postgres"] });
    settingsMocks.updateConfig.mockResolvedValue({ success: true, config, message: "saved" });
    settingsMocks.exportConfig.mockResolvedValue(new Blob(["{}"], { type: "application/json" }));
    settingsMocks.importConfig.mockResolvedValue({ success: true, config, message: "imported" });
    for (const fn of [settingsMocks.setCheckEnabled, settingsMocks.setCheckAutoHeal, settingsMocks.bulkUpdateChecks, settingsMocks.addScenario, settingsMocks.removeScenario, settingsMocks.setScenarioCritical, settingsMocks.addResource, settingsMocks.removeResource]) {
      fn.mockResolvedValue({ success: true });
    }
    vi.stubGlobal("confirm", vi.fn(() => true));
    vi.stubGlobal("alert", vi.fn());
    URL.createObjectURL = vi.fn(() => "blob:test");
    URL.revokeObjectURL = vi.fn();
  });

  it("edits general settings, saves, and switches auto-refresh", async () => {
    renderWithProviders(<SettingsDialog isOpen onClose={vi.fn()} autoRefresh onAutoRefreshChange={vi.fn()} />);
    expect(await screen.findByTestId("settings-dialog")).toBeInTheDocument();
    await screen.findByText("Grace Period");
    const spinbuttons = screen.getAllByRole("spinbutton");
    expect(spinbuttons.length).toBeGreaterThan(0);
    const gracePeriod = spinbuttons[0];
    if (!gracePeriod) throw new Error("Grace period input was not rendered");
    fireEvent.change(gracePeriod, { target: { value: "90" } });
    fireEvent.click(screen.getByRole("switch", { name: /disable auto-refresh/i }));
    fireEvent.click(screen.getByTestId("settings-save"));
    await waitFor(() => expect(settingsMocks.updateConfig).toHaveBeenCalled());
  });

  it("covers health check controls and category collapse", async () => {
    renderWithProviders(<SettingsDialog isOpen onClose={vi.fn()} autoRefresh={false} onAutoRefreshChange={vi.fn()} />);
    await screen.findByTestId("settings-dialog");
    fireEvent.click(screen.getByRole("button", { name: "Health Checks" }));
    await screen.findByText("DNS");
    fireEvent.click(screen.getByRole("button", { name: /enable all$/i }));
    fireEvent.click(screen.getByRole("button", { name: /disable all$/i }));
    fireEvent.click(screen.getByRole("button", { name: /enable all auto-heal/i }));
    fireEvent.click(screen.getByRole("button", { name: /disable all auto-heal/i }));
    fireEvent.click(screen.getByRole("button", { name: /infrastructure.*1 checks/i }));
    await waitFor(() => expect(settingsMocks.bulkUpdateChecks).toHaveBeenCalledTimes(4));
  });

  it("manages monitoring and import/export tabs", async () => {
    renderWithProviders(<SettingsDialog isOpen onClose={vi.fn()} autoRefresh onAutoRefreshChange={vi.fn()} />);
    await screen.findByTestId("settings-dialog");
    fireEvent.click(screen.getByRole("button", { name: "Monitoring" }));
    await screen.findByText("Monitoring Configuration");
    const inputs = screen.getAllByRole("textbox");
    const scenarioInput = inputs[0];
    const resourceInput = inputs[1];
    const addButtons = screen.getAllByRole("button", { name: /^add$/i });
    const removeButton = screen.getAllByTitle(/remove/i)[0];
    if (!scenarioInput || !resourceInput || !addButtons[0] || !addButtons[1] || !removeButton) {
      throw new Error("Monitoring controls were not rendered");
    }
    fireEvent.change(scenarioInput, { target: { value: "new-scenario" } });
    fireEvent.click(addButtons[0]);
    fireEvent.change(resourceInput, { target: { value: "qdrant" } });
    fireEvent.click(addButtons[1]);
    fireEvent.click(removeButton);
    fireEvent.click(screen.getByRole("button", { name: /import \/ export/i }));
    fireEvent.click(screen.getByRole("button", { name: /export configuration/i }));
    fireEvent.click(screen.getByRole("button", { name: /reset to defaults/i }));
    expect(settingsMocks.exportConfig).toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: /cancel/i }));
  });
});
