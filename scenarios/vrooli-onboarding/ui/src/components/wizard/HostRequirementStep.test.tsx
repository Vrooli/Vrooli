import { fireEvent, screen, waitFor } from "../../test-utils";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { vi } from "vitest";
import { HostRequirementStep } from "./HostRequirementStep";

const hostApi = vi.hoisted(() => ({ fetchHostRequirements: vi.fn(), fetchTargets: vi.fn() }));
vi.mock("../../api/host", () => hostApi);

describe("HostRequirementStep", () => {
  it("renders manifest config and emits a typed safeguard patch", async () => {
    hostApi.fetchHostRequirements.mockResolvedValue({
      tools: [],
      safeguards: [{
        name: "firewall", required: false, status: "optional", reason: "Network safety", risk: "high",
        config_schema: { type: "object", properties: { mode: { type: "string", enum: ["audit", "enforce"] }, retries: { type: "integer" }, advanced: { type: "object", title: "Advanced", description: "Provider settings" } } },
      }],
    });
    const onSafeguard = vi.fn();
    const onConfig = vi.fn();
    renderWithProviders(<HostRequirementStep onTool={vi.fn()} onSafeguard={onSafeguard} onHostConfig={onConfig} />);
    expect(await screen.findByTestId("risk-indicator")).toHaveTextContent("high");
    fireEvent.click(screen.getByRole("checkbox", { name: /firewall/i }));
    fireEvent.click(screen.getByRole("button", { name: "mode" }));
    fireEvent.click(screen.getByRole("option", { name: "enforce" }));
    fireEvent.change(screen.getByLabelText("retries"), { target: { value: "2" } });
    expect(screen.getByText(/The object configuration field cannot be edited in this wizard/)).toBeInTheDocument();
    expect(onSafeguard).toHaveBeenCalledWith("firewall", true);
    expect(onConfig).toHaveBeenLastCalledWith("host_safeguards", "firewall", { mode: "enforce", retries: 2 });
  });

  it("shows remote requirements read-only and blocks unsupported host mutations", async () => {
    hostApi.fetchTargets.mockResolvedValue({ targets: [{ id: "node-1", kind: "NODE_KIND_AGENT", available: true }] });
    hostApi.fetchHostRequirements.mockResolvedValue({
      tools: [],
      safeguards: [{
        name: "firewall", required: false, status: "optional", reason: "Network safety",
        config_schema: { type: "object", properties: { mode: { type: "string", enum: ["audit", "enforce"] } } },
      }],
    });
    const onSafeguard = vi.fn();
    const onConfig = vi.fn();
    renderWithProviders(<HostRequirementStep target="node-1" onTool={vi.fn()} onSafeguard={onSafeguard} onHostConfig={onConfig} />);

    expect(await screen.findByTestId("remote-host-read-only")).toHaveTextContent("Remote host requirements are read-only here");
    expect(hostApi.fetchHostRequirements).toHaveBeenCalledWith("node-1");
    expect(screen.getByRole("checkbox", { name: /firewall/i })).toBeDisabled();
    expect(screen.queryByTestId("config-firewall")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("requirement-entry"));
    expect(onSafeguard).not.toHaveBeenCalled();
    expect(onConfig).not.toHaveBeenCalled();
  });

  it("retains the local owner's saved settings across a remote target loss", async () => {
    const localRequirements = {
      tools: [],
      safeguards: [{ name: "firewall", required: false, status: "opted_in", reason: "Network safety", config: { region: "saved-region" }, config_schema: { type: "object", properties: { region: { type: "string", title: "Region" } } } }],
    };
    hostApi.fetchHostRequirements.mockImplementation(async (target: string) => {
      if (target === "node-1") throw new Error("Bridge unavailable");
      return localRequirements;
    });
    const onSafeguard = vi.fn();
    const onConfig = vi.fn();
    const view = renderWithProviders(<HostRequirementStep target="local" onTool={vi.fn()} onSafeguard={onSafeguard} onHostConfig={onConfig} />);

    expect(await screen.findByLabelText("Region")).toHaveValue("saved-region");
    expect(screen.getByRole("checkbox", { name: /firewall/i })).toBeChecked();
    view.rerender(<HostRequirementStep target="node-1" onTool={vi.fn()} onSafeguard={onSafeguard} onHostConfig={onConfig} />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Bridge unavailable");
    expect(screen.queryByTestId("requirement-entry")).not.toBeInTheDocument();
    expect(onSafeguard).not.toHaveBeenCalled();
    expect(onConfig).not.toHaveBeenCalled();

    view.rerender(<HostRequirementStep target="local" onTool={vi.fn()} onSafeguard={onSafeguard} onHostConfig={onConfig} />);
    await waitFor(() => expect(screen.getByLabelText("Region")).toHaveValue("saved-region"));
    expect(screen.getByRole("checkbox", { name: /firewall/i })).toBeChecked();
  });

  it("does not load or expose edits for an unsupported empty target", async () => {
    hostApi.fetchHostRequirements.mockClear();
    hostApi.fetchTargets.mockClear();
    renderWithProviders(<HostRequirementStep target="" onTool={vi.fn()} onSafeguard={vi.fn()} />);
    expect(await screen.findByTestId("unsupported-host-target")).toHaveTextContent("Choose the local host or an available remote node");
    expect(hostApi.fetchHostRequirements).not.toHaveBeenCalled();
    expect(hostApi.fetchTargets).not.toHaveBeenCalled();
    expect(screen.queryByTestId("requirement-entry")).not.toBeInTheDocument();
  });

  it("rejects unknown IDs and non-host node kinds before reading host requirements", async () => {
    hostApi.fetchHostRequirements.mockClear();
    hostApi.fetchTargets.mockResolvedValue({ targets: [{ id: "phone-1", kind: "NODE_KIND_ATTACHED" }] });
    const view = renderWithProviders(<HostRequirementStep target="unknown" onTool={vi.fn()} onSafeguard={vi.fn()} />);
    expect(await screen.findByTestId("unsupported-host-target")).toHaveTextContent("not a registered host");
    expect(hostApi.fetchHostRequirements).not.toHaveBeenCalled();
    view.rerender(<HostRequirementStep target="phone-1" onTool={vi.fn()} onSafeguard={vi.fn()} />);
    expect(await screen.findByTestId("unsupported-host-target")).toHaveTextContent("not a registered host");
    expect(hostApi.fetchHostRequirements).not.toHaveBeenCalled();
  });
});
