import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";

const ensureWorkspace = vi.hoisted(() => vi.fn());
const fetchDiagnostics = vi.hoisted(() => vi.fn());
vi.mock("../api/workspace", () => ({ ensureWorkspace }));
vi.mock("../api/diagnostics", () => ({ fetchDiagnostics }));

import { SettingsPage } from "./SettingsPage";

describe("Settings data health", () => {
  beforeEach(() => { vi.clearAllMocks(); ensureWorkspace.mockResolvedValue({ id: "w1" }); });

  it("shows an empty actionable state", async () => {
    fetchDiagnostics.mockResolvedValue({ workspaceId: "w1", generatedAt: "", schemaVersion: 0, databaseOk: true, providers: [], imageGeneration: { capability: "unknown", reason: "service missing", permission: "off", dispatchAllowed: false, quoteAvailable: false }, findings: [] });
    renderWithProviders(<SettingsPage />);
    await waitFor(() => expect(screen.getByText(/No material data-health findings/)).toBeInTheDocument());
    expect(screen.getByText("Image Tools capability: unknown")).toBeInTheDocument();
    expect(screen.getByText(/Permission: Off/)).toHaveTextContent(/dispatch is disabled/i);
  });

  it("shows findings and provider states", async () => {
    fetchDiagnostics.mockResolvedValue({ workspaceId: "w1", generatedAt: "", schemaVersion: 0, databaseOk: true, providers: [{ name: "ai_gateway", state: "not_configured", reason: "manual fallback" }], imageGeneration: { capability: "available", reason: "local capability available; quote unavailable", permission: "off", dispatchAllowed: false, quoteAvailable: false }, findings: [{ code: "old_prices", severity: "info", count: 2, message: "Old prices", action: "Review prices" }] });
    renderWithProviders(<SettingsPage />);
    await waitFor(() => expect(screen.getByText("Old prices (2)")).toBeInTheDocument());
    expect(screen.getByText("ai_gateway")).toBeInTheDocument();
    expect(screen.getByText(/not_configured/)).toBeInTheDocument();
    expect(screen.getByText("Image Tools capability: available")).toBeInTheDocument();
  });

  it("reports data-health loading failures", async () => {
    fetchDiagnostics.mockRejectedValue(new Error("diagnostics unavailable"));
    renderWithProviders(<SettingsPage />);
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("diagnostics unavailable"));
  });
});
