import { fireEvent, render, screen } from "@/test-utils";
import { afterAll, beforeEach, describe, expect, it, vi } from "vitest";
import App from "../../App";
import { usePipelineStore } from "../../store";

const originalSetScenario = usePipelineStore.getState().setScenario;

vi.mock("../../components/scenario-inventory", () => ({
  ScenarioInventory: ({ onScenarioLaunch }: { onScenarioLaunch: (scenario: { name: string }) => void }) => (
    <button type="button" onClick={() => { onScenarioLaunch({ name: "canvas-lab" }); }}>
      Launch canvas-lab
    </button>
  ),
}));
vi.mock("../../pages", () => ({
  GeneratorPage: ({ scenarioName, onOpenSigningTab }: { scenarioName: string; onOpenSigningTab: (name: string) => void }) => (
    <div>Generator for {scenarioName}<button type="button" onClick={() => { onOpenSigningTab(scenarioName); }}>Open signing</button></div>
  ),
}));
vi.mock("../../components/docs/DocsPanel", () => ({ DocsPanel: () => <div>Docs panel</div> }));
vi.mock("../../components/signing", () => ({ SigningPage: ({ initialScenario }: { initialScenario: string }) => <div>Signing {initialScenario}</div> }));
vi.mock("../../components/scenario-inventory/RecordsManager", () => ({
  RecordsManager: ({ onEditSigning }: { onEditSigning: (name: string) => void }) => (
    <button type="button" onClick={() => { onEditSigning("canvas-lab"); }}>Edit canvas signing</button>
  ),
}));
vi.mock("../../components/livedesktop", () => ({ LiveDesktopDrawer: () => null }));
vi.mock("../../components/captures", () => ({ CapturesDrawer: () => null }));
vi.mock("../../hooks/useServerSync", () => ({ useServerSync: () => undefined }));

describe("App navigation", () => {
  beforeEach(() => {
    localStorage.clear();
    usePipelineStore.getState().reset();
    usePipelineStore.setState({ setScenario: vi.fn() });
  });

  afterAll(() => {
    usePipelineStore.setState({ setScenario: originalSetScenario });
  });

  it("moves an inventory selection into its generator and signing workflow", () => {
    render(<App />);
    fireEvent.click(screen.getByRole("button", { name: "Launch canvas-lab" }));
    expect(screen.getByText("Generator for canvas-lab")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Open signing" }));
    expect(screen.getByText("Signing canvas-lab")).toBeInTheDocument();
  });

  it("switches among Docs, Apps, and Signing tabs", () => {
    render(<App />);
    fireEvent.click(screen.getByRole("tab", { name: "Docs" }));
    expect(screen.getByText("Docs panel")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: "Apps" }));
    fireEvent.click(screen.getByRole("button", { name: "Edit canvas signing" }));
    expect(screen.getByText("Signing canvas-lab")).toBeInTheDocument();
  });

  it("shows the active build action bar only for generator view", () => {
    usePipelineStore.setState({ pipelineId: "pipe-1", runStatus: "completed" });
    render(<App />);
    fireEvent.click(screen.getByRole("tab", { name: "Generate" }));
    expect(screen.getByText("Build ready - spawn an agent to verify or improve")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: "Docs" }));
    expect(screen.queryByText("Build ready - spawn an agent to verify or improve")).not.toBeInTheDocument();
  });
});
