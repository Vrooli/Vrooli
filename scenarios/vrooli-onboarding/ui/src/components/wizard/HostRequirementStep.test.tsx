import { fireEvent, screen } from "../../test-utils";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { vi } from "vitest";
import { HostRequirementStep } from "./HostRequirementStep";

const hostApi = vi.hoisted(() => ({ fetchHostRequirements: vi.fn() }));
vi.mock("../../api/host", () => hostApi);

describe("HostRequirementStep", () => {
  it("renders manifest config and emits a typed safeguard patch", async () => {
    hostApi.fetchHostRequirements.mockResolvedValue({
      tools: [],
      safeguards: [{
        name: "firewall", required: false, status: "optional", reason: "Network safety", risk: "high",
        config_schema: { type: "object", properties: { mode: { type: "string", enum: ["audit", "enforce"] }, retries: { type: "integer" } } },
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
    expect(onSafeguard).toHaveBeenCalledWith("firewall", true);
    expect(onConfig).toHaveBeenLastCalledWith("host_safeguards", "firewall", { mode: "enforce", retries: 2 });
  });
});
