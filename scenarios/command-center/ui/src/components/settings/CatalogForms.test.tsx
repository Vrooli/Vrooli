import { describe, expect, it, vi } from "vitest";
import { fireEvent } from "@testing-library/react";
import { renderWithProviders, screen } from "../../test-utils/renderWithProviders";
import type { CatalogEntry, Catalogs } from "../../lib/api";
import { AdvancedJson, CatalogForm } from "./CatalogForms";

const catalogs: Catalogs = {
  themes: [{ id: "cosmos", label: "Cosmos", tokens: { "--color-primary": "#8aa8ff", "--color-background": "rgba(6,4,26,1)" }, cornerRadius: "2px", gradient: "radial-gradient(#000)" }],
  compositions: [{ id: "orbital-field", slots: { running: { shape: "scalar", role: "primary", whenUnbound: "decorative" }, healthy: { shape: "scalar" } } }, { id: "flat", slots: {} }, { id: "shapeless", slots: { any: {} } }],
  signals: [
    { id: "active_scenarios", label: "Apps running", shape: "scalar", coverage: "NOW" },
    { id: "funnel_30d", label: "Funnel", shape: "rows", coverage: "IN-REACH" },
  ],
  connectors: [{ id: "swarm-manager", enabled: true, transport: "http-json", baseUrlEnv: "SWARM_MANAGER_BASE_URL", paths: ["/api/v1/stats"], pack: true }],
  rooms: [{ id: "mission-control", title: "Mission Control", theme: "cosmos", composition: "orbital-field", bind: { running: "active_scenarios" }, beats: [{ hero: "active_scenarios", dwellSeconds: 12 }] }],
  readouts: [{ id: "scalar" }],
} as Catalogs;

const form = (catalog: string, draft: CatalogEntry, patch = vi.fn()) => {
  renderWithProviders(<CatalogForm catalog={catalog} draft={draft} patch={patch} catalogs={catalogs} blockingError={null} />);
  return patch;
};

describe("CatalogForm — rooms", () => {
  it("renders the room fields and edits the title through patch", () => {
    const patch = form("rooms", catalogs.rooms![0]!);
    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Ops" } });
    expect(patch).toHaveBeenCalledWith({ title: "Ops" });
  });

  it("binds a slot to a shape-compatible signal and clears it when unbound", () => {
    const patch = form("rooms", catalogs.rooms![0]!);
    const runningSlot = screen.getByLabelText("running · scalar", { exact: false });
    fireEvent.change(runningSlot, { target: { value: "active_scenarios" } });
    expect(patch).toHaveBeenCalledWith({ bind: { running: "active_scenarios" } });
    fireEvent.change(runningSlot, { target: { value: "" } });
    expect(patch).toHaveBeenLastCalledWith({ bind: {} });
  });

  it("shows a hint when the composition declares no slots", () => {
    form("rooms", { id: "r", composition: "flat" });
    expect(screen.getByText("This composition declares no slots.")).toBeInTheDocument();
  });

  it("summarises beats read-only", () => {
    form("rooms", catalogs.rooms![0]!);
    expect(screen.getByText("active_scenarios · 12s")).toBeInTheDocument();
  });
});

describe("CatalogForm — themes, connectors, signals", () => {
  it("edits a theme token colour", () => {
    const patch = form("themes", catalogs.themes![0]!);
    fireEvent.change(screen.getByLabelText("Primary colour"), { target: { value: "#112233" } });
    expect(patch).toHaveBeenCalledWith({ tokens: { "--color-primary": "#112233", "--color-background": "rgba(6,4,26,1)" } });
  });

  it("renders a text swatch for a non-hex token", () => {
    form("themes", catalogs.themes![0]!);
    // background is rgba(...), so it uses the text input path, not a colour picker.
    expect(screen.getByLabelText("Background")).toHaveValue("rgba(6,4,26,1)");
  });

  it("toggles a connector and rewrites its paths", () => {
    const patch = form("connectors", catalogs.connectors![0]!);
    fireEvent.click(screen.getByLabelText("Enabled", { exact: false }));
    expect(patch).toHaveBeenCalledWith({ enabled: false });
    fireEvent.change(screen.getByLabelText("Paths", { exact: false }), { target: { value: "/a\n/b\n" } });
    expect(patch).toHaveBeenCalledWith({ paths: ["/a", "/b"] });
  });

  it("edits a signal coverage", () => {
    const patch = form("signals", catalogs.signals![0]!);
    fireEvent.change(screen.getByLabelText("Coverage"), { target: { value: "MISSING" } });
    expect(patch).toHaveBeenCalledWith({ coverage: "MISSING" });
  });

  it("leans on advanced JSON for structural catalogs", () => {
    form("compositions", catalogs.compositions![0]!);
    expect(screen.getByText(/Compositions declare slots/)).toBeInTheDocument();
  });

  it("surfaces a blocking error when supplied", () => {
    renderWithProviders(<CatalogForm catalog="rooms" draft={catalogs.rooms![0]!} patch={vi.fn()} catalogs={catalogs} blockingError={"bad binding"} />);
    expect(screen.getByRole("alert")).toHaveTextContent("bad binding");
  });
});

describe("CatalogForm — fallback and default branches", () => {
  it("keeps an out-of-catalog room selection visible and renders no beats block when empty", () => {
    form("rooms", { id: "r", title: "R", theme: "ghost-theme", composition: "orbital-field", bind: {}, beats: [] });
    // The unknown theme is preserved as a selectable option rather than silently dropped.
    expect(screen.getByRole("option", { name: "ghost-theme" })).toBeInTheDocument();
    expect(screen.queryByText(/Beats/)).not.toBeInTheDocument();
  });

  it("applies connector defaults when transport, paths, and pack are absent", () => {
    const patch = form("connectors", { id: "bare", enabled: false });
    expect(screen.queryByLabelText(/Connector pack/)).not.toBeInTheDocument();
    expect(screen.getByLabelText("Transport")).toHaveValue("http-json");
    fireEvent.change(screen.getByLabelText("Transport"), { target: { value: "connect" } });
    expect(patch).toHaveBeenCalledWith({ transport: "connect" });
  });

  it("shows an unknown shape hint for a signal missing its shape", () => {
    form("signals", { id: "s", label: "S" });
    expect(screen.getByText(/unknown/)).toBeInTheDocument();
  });

  it("renders the readout structural hint", () => {
    form("readouts", { id: "scalar" });
    expect(screen.getByText(/Readouts map a signal kind/)).toBeInTheDocument();
  });

  it("labels a shapeless slot as accepting any shape and offers every signal", () => {
    form("rooms", { id: "r3", composition: "shapeless", bind: {} });
    expect(screen.getByLabelText("any · any", { exact: false })).toBeInTheDocument();
  });

  it("degrades to empty option lists when the catalogs are empty", () => {
    renderWithProviders(<CatalogForm catalog="rooms" draft={{ id: "x", composition: "missing" }} patch={vi.fn()} catalogs={{}} blockingError={null} />);
    expect(screen.getByText("This composition declares no slots.")).toBeInTheDocument();
  });
});

describe("AdvancedJson", () => {
  it("replaces the draft on valid JSON and rejects invalid or id-less input", () => {
    const onReplace = vi.fn();
    renderWithProviders(<AdvancedJson draft={{ id: "x", title: "A" }} onReplace={onReplace} />);
    const editor = screen.getByLabelText("Advanced JSON editor");

    fireEvent.change(editor, { target: { value: '{"id":"x","title":"B"}' } });
    expect(onReplace).toHaveBeenCalledWith({ id: "x", title: "B" });

    onReplace.mockClear();
    fireEvent.change(editor, { target: { value: "{ not json" } });
    expect(onReplace).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toHaveTextContent("Invalid JSON");

    onReplace.mockClear();
    fireEvent.change(editor, { target: { value: '{"title":"no id"}' } });
    expect(onReplace).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toHaveTextContent("string id");
  });
});
