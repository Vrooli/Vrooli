import { beforeEach, describe, expect, it, vi } from "vitest";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import SettingsPage from "./SettingsPage";
import type { Catalogs } from "../lib/api";

const catalogs: Catalogs = {
  rooms: [{ id: "mission-control", title: "Mission Control", theme: "cosmos", composition: "orbital-field", bind: { running: "active_scenarios" } }],
  themes: [{ id: "cosmos", label: "Cosmos", tokens: { "--color-primary": "#00d4ff" } }],
  compositions: [{ id: "orbital-field", slots: { running: { shape: "scalar" }, healthy: { shape: "scalar" } } }],
  signals: [{ id: "active_scenarios", label: "Active scenarios", shape: "scalar", coverage: "NOW", sample: { value: 12, series: [], basis: "sample" } }],
  connectors: [{ id: "swarm", enabled: true, transport: "connect" }],
  readouts: [{ id: "scalar-figure" }],
};

vi.mock("../lib/api", () => ({
  fetchCatalogs: vi.fn(() => Promise.resolve(catalogs)),
  fetchBoardSettings: vi.fn(() => Promise.resolve({ cycleSeconds: 60, transition: "crossfade", rooms: [{ id: "mission-control", enabled: true }] })),
  saveBoardSettings: vi.fn(() => Promise.resolve({ cycleSeconds: 60, transition: "crossfade", rooms: [] })),
  saveCatalogEntry: vi.fn(() => Promise.resolve({ id: "x" })),
  hasValue: (reading: { value?: unknown }) => reading.value !== null && reading.value !== undefined,
}));

const renderAt = (path: string) => renderWithProviders(<MemoryRouter initialEntries={[path]}><SettingsPage /></MemoryRouter>);

describe("SettingsPage — section-model surface", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders the five plain-language destinations", async () => {
    renderAt("/settings");
    expect(await screen.findByText("Shape the board")).toBeInTheDocument();
    for (const label of ["Rooms", "Looks", "Signals & sources", "Board settings", "Operator mode"]) expect(screen.getByText(label)).toBeInTheDocument();
  });

  it("opens Rooms on the real room model", async () => {
    renderAt("/settings?destination=rooms");
    expect(await screen.findByTestId("settings-rooms")).toBeInTheDocument();
    expect(screen.getByText(/sections that cycle/)).toBeInTheDocument();
  });

  it("shows the operator catalog rail and can switch catalogs", async () => {
    renderAt("/settings?mode=catalogs");
    expect(await screen.findByText("All catalogs")).toBeInTheDocument();
    const rail = screen.getByRole("navigation", { name: "Catalogs" });
    expect(rail).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /themes/i }));
    expect(await screen.findByText(/A theme is the paint/)).toBeInTheDocument();
  });

  it("opens Board settings with persistence controls", async () => {
    renderAt("/settings?destination=board");
    expect(await screen.findByTestId("settings-board")).toBeInTheDocument();
    expect(screen.getByText("Default dwell")).toBeInTheDocument();
  });
});
