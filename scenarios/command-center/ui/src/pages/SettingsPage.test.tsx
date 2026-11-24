import { beforeEach, describe, expect, it, vi } from "vitest";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import SettingsPage from "./SettingsPage";
import { fetchBoardSettings, saveBoardSettings, type Catalogs } from "../lib/api";

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
    expect(screen.getByRole("navigation", { name: "Customization destinations" })).toBeInTheDocument();
    for (const label of ["Rooms", "Looks", "Signals & sources", "Board settings", "Operator mode"]) expect(screen.getByText(label)).toBeInTheDocument();
  });

  it("opens Rooms on the real room model", async () => {
    renderAt("/settings?destination=rooms");
    expect(await screen.findByTestId("settings-rooms")).toBeInTheDocument();
    expect(screen.getByText(/sections that cycle/)).toBeInTheDocument();
  });

  it("opens section controls immediately after adding a section to an ambient room", async () => {
    renderAt("/settings?destination=rooms&room=mission-control");
    expect(await screen.findByText("Ambient room")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Add section" }));
    expect(await screen.findByText("Hero")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save room" })).toBeInTheDocument();
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

  it("persists a room toggle when board settings have no explicit room entries", async () => {
    vi.mocked(fetchBoardSettings).mockResolvedValueOnce({ cycleSeconds: 60, transition: "crossfade", rooms: [] });
    renderAt("/settings?destination=board");
    const roomToggle = await screen.findByRole("checkbox", { name: "Mission Control" });
    expect(roomToggle).toBeChecked();

    await userEvent.click(roomToggle);
    expect(roomToggle).not.toBeChecked();
    await userEvent.click(screen.getByRole("button", { name: "Save board settings" }));

    expect(saveBoardSettings).toHaveBeenCalledWith({
      cycleSeconds: 60,
      transition: "crossfade",
      rooms: [{ id: "mission-control", enabled: false }],
    });
  });

  it("gives the Signals catalog a navigable table with row and column headers", async () => {
    renderAt("/settings?destination=signals");
    expect(await screen.findByRole("table", { name: "Configured signals and their measurement and source details" })).toBeInTheDocument();
    expect(screen.getByRole("columnheader", { name: "Signal" })).toBeInTheDocument();
    expect(screen.getByRole("columnheader", { name: "Measurement" })).toBeInTheDocument();
    expect(screen.getByRole("rowheader", { name: "Active scenarios" })).toBeInTheDocument();
    expect(screen.getByRole("cell", { name: "a single number" })).toBeInTheDocument();
    expect(screen.getByText("Swipe horizontally to see readout and source details.")).toBeInTheDocument();
  });

  it("keeps Return to board reachable by keyboard", async () => {
    renderAt("/settings?destination=signals");
    const user = userEvent.setup();
    await screen.findByRole("button", { name: "Return to board" });
    await user.tab();
    await user.tab();
    await user.keyboard("{Enter}");
    expect(await screen.findByText("Shape the board")).toBeInTheDocument();
  });
});
