import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { DeploymentModal } from "./DeploymentModal";

describe("DeploymentModal", () => {
  const onChange = vi.fn();
  const onClose = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  function renderModal(
    overrides: Partial<ComponentProps<typeof DeploymentModal>> = {},
  ) {
    return renderWithProviders(
      <DeploymentModal
        open
        deploymentMode="external-server"
        serverType="external"
        allowedServerTypes={["external", "static", "node", "executable"]}
        onChange={onChange}
        onClose={onClose}
        {...overrides}
      />,
    );
  }

  it("keeps cloud deployment unavailable while allowing an operator to choose bundled mode", async () => {
    const user = userEvent.setup();
    renderModal();

    const cloud = screen.getByRole("button", {
      name: /Cloud API bundle/,
    });
    expect(cloud).toBeDisabled();

    await user.click(
      screen.getByRole("button", { name: /Fully bundled\/offline/ }),
    );
    expect(onChange).toHaveBeenCalledWith("bundled", "external");
  });

  it("prevents unavailable server types and selects an allowed data source", async () => {
    const user = userEvent.setup();
    renderModal({ allowedServerTypes: ["external", "static"] });

    const node = screen.getByRole("button", { name: /Embedded Node server/ });
    expect(node).toBeDisabled();

    await user.click(
      screen.getByRole("button", { name: /Static files \(UI only\)/ }),
    );
    expect(onChange).toHaveBeenCalledWith("external-server", "static");
  });

  it("shows guidance and closes through the explicit control", async () => {
    const user = userEvent.setup();
    renderModal();

    await user.click(screen.getByRole("button", { name: "Help me decide" }));
    expect(screen.getByText("Quick guidance")).toBeInTheDocument();
    expect(
      screen.getByText(/experimental\) and prepare a bundle manifest/),
    ).toBeInTheDocument();

    await user.click(
      screen.getByRole("button", { name: "Close deployment dialog" }),
    );
    expect(onClose).toHaveBeenCalledOnce();
  });
});
