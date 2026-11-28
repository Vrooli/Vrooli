import "@testing-library/jest-dom";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithProviders } from "./test-utils/renderWithProviders";

const api = vi.hoisted(() => ({ listScenarios: vi.fn(), getScenarioDependencies: vi.fn(), checkReachability: vi.fn() }));
vi.mock("./lib/api", () => api);

import { StepManifest } from "./components/wizard/StepManifest";

const manifest = { scenario: { id: "demo" }, target: { vps: { host: "203.0.113.10", port: 22, user: "root" } }, edge: { domain: "example.com", caddy: { enabled: true, email: "" } }, ports: { ui: 3000, ws: 3001 }, bundle: { include_packages: true, include_autoheal: true }, dependencies: { resources: [], scenarios: ["demo"] } };
const deployment = {
  manifestJson: JSON.stringify(manifest, null, 2), setManifestJson: vi.fn(), parsedManifest: { ok: true, value: manifest },
  validationIssues: [{ severity: "warning", path: "edge.domain", message: "Check DNS", hint: "Use a real domain" }], validationError: null, isValidating: false,
  validate: vi.fn(), normalizedManifest: manifest, applyAllFixes: vi.fn(), undo: vi.fn(), redo: vi.fn(), canUndo: true, canRedo: true,
  resetManifestToDefaults: vi.fn(), resetManifestWithScenario: vi.fn(), sshKeyPath: null, setSSHKeyPath: vi.fn(), setSSHConnectionStatus: vi.fn(),
};

describe("manifest editor workflow", () => {
  beforeEach(() => {
    api.listScenarios.mockResolvedValue({ scenarios: [{ id: "demo", displayName: "Demo Scenario", description: "A demo", ports: { ui: { port: 3000, description: "UI" }, websocket: { range: "3001-3010" } } }] });
    api.getScenarioDependencies.mockResolvedValue({ source: "analyzer", resources: ["postgres"], scenarios: ["demo"] });
    api.checkReachability.mockResolvedValue({ results: [{ type: "host", reachable: true, message: "reachable" }, { type: "domain", reachable: true, message: "resolves" }] });
  });

  it("covers form editing, scenario selection, dependency hydration, resets, validation, and JSON mode", async () => {
    renderWithProviders(<StepManifest deployment={deployment as any} />);
    await waitFor(() => expect(screen.getByText("Demo Scenario")).toBeInTheDocument());
    fireEvent.focus(screen.getByPlaceholderText("Search scenarios..."));
    fireEvent.click(screen.getByRole("button", { name: /demo/i }));
    await waitFor(() => expect(api.getScenarioDependencies).toHaveBeenCalledWith("demo"));
    fireEvent.change(screen.getByLabelText("Host IP or Hostname"), { target: { value: "203.0.113.11" } });
    fireEvent.change(screen.getByLabelText("Domain"), { target: { value: "cloud.example" } });
    fireEvent.change(screen.getByLabelText("Let's Encrypt Email"), { target: { value: "ops@example.com" } });
    fireEvent.click(screen.getByLabelText("Include packages"));
    fireEvent.click(screen.getByLabelText("Include autoheal"));
    fireEvent.click(screen.getByLabelText("Enable Caddy"));
    fireEvent.click(screen.getByRole("button", { name: "Reset" }));
    fireEvent.click(screen.getByRole("button", { name: /Reset with current scenario/ }));
    fireEvent.click(screen.getByRole("button", { name: "Reset" }));
    fireEvent.click(screen.getByRole("button", { name: /Reset to defaults/ }));
    fireEvent.click(screen.getByRole("button", { name: "Revalidate" }));
    expect(deployment.validate).toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "JSON" }));
    expect(screen.getByTestId("manifest-input")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Form" }));
    fireEvent.keyDown(window, { key: "z", ctrlKey: true });
    fireEvent.keyDown(window, { key: "z", ctrlKey: true, shiftKey: true });
    fireEvent.keyDown(window, { key: "y", ctrlKey: true });
  });

  it("renders invalid JSON, API validation errors, and blocking validation issues", () => {
    renderWithProviders(<StepManifest deployment={{
      ...deployment,
      parsedManifest: { ok: false, error: "Unexpected token" },
      validationIssues: [
        { severity: "error", path: "target.host", message: "Host is required" },
        { severity: "warning", path: "edge.domain", message: "Domain warning", hint: "Check DNS" },
      ],
      validationError: "Validation service unavailable",
      isValidating: true,
      normalizedManifest: null,
    } as any} />);
    expect(screen.getByText("Invalid JSON")).toBeInTheDocument();
    expect(screen.getByText("Validating...")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "JSON" }));
    expect(screen.getByTestId("manifest-input")).toBeInTheDocument();
    expect(screen.getByText("Validation service unavailable")).toBeInTheDocument();
  });

  it("handles unavailable scenario metadata, unknown scenarios, reachability failures, and optional fields", async () => {
    api.listScenarios.mockResolvedValue({ scenarios: [{ id: "other", description: "No ports" }] });
    api.getScenarioDependencies.mockRejectedValue(new Error("dependency analyzer unavailable"));
    api.checkReachability.mockResolvedValue({
      results: [{ type: "host", reachable: false, message: "host offline", hint: "Check SSH" }],
    });
    const sparseManifest = {
      ...manifest,
      scenario: { id: "missing" },
      target: { vps: { host: "198.51.100.20", port: 2222, user: "deployer" } },
      edge: { domain: "missing.example", caddy: { enabled: true, email: "" } },
      ports: {},
      bundle: { include_packages: false, include_autoheal: false },
    };
    const firstRender = renderWithProviders(<StepManifest deployment={{ ...deployment, manifestJson: JSON.stringify(sparseManifest), parsedManifest: { ok: true, value: sparseManifest }, validationIssues: [], normalizedManifest: sparseManifest } as any} />);
    await waitFor(() => expect(screen.getByText(/Scenario "missing" not found/)).toBeInTheDocument());
    expect(screen.getByText("No ports defined in this scenario's service.json")).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText("Enable Caddy"));
    const caddyDisabled = { ...sparseManifest, edge: { ...sparseManifest.edge, caddy: { enabled: false, email: "" } } };
    firstRender.rerender(<StepManifest deployment={{ ...deployment, manifestJson: JSON.stringify(caddyDisabled), parsedManifest: { ok: true, value: caddyDisabled }, validationIssues: [], normalizedManifest: caddyDisabled } as any} />);
    expect(screen.queryByLabelText("Let's Encrypt Email")).not.toBeInTheDocument();
    fireEvent.focus(screen.getByPlaceholderText("Search scenarios..."));
    fireEvent.click(screen.getByRole("button", { name: /other/i }));
    await waitFor(() => expect(api.getScenarioDependencies).toHaveBeenCalledWith("other"));
    await new Promise((resolve) => setTimeout(resolve, 1100));
    expect(screen.getByText("Check SSH")).toBeInTheDocument();

    api.listScenarios.mockRejectedValue(new Error("scenario registry unavailable"));
    firstRender.unmount();
    const { unmount } = renderWithProviders(<StepManifest deployment={deployment as any} />);
    await waitFor(() => expect(api.listScenarios).toHaveBeenCalled());
    fireEvent.focus(screen.getByPlaceholderText("Search scenarios..."));
    expect(screen.getByText("scenario registry unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Reset" }));
    fireEvent.mouseDown(document.body);
    expect(screen.queryByText("Reset to defaults")).not.toBeInTheDocument();
    unmount();
  });
});
