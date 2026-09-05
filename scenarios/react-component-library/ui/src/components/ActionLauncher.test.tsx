import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

const { startWorkflow, listScenarios, listCatalogAssets } = vi.hoisted(() => ({
  startWorkflow: vi.fn(),
  listScenarios: vi.fn(),
  listCatalogAssets: vi.fn(),
}));

vi.mock("../api/workflows", () => ({ workflowsClient: { startWorkflow } }));
vi.mock("../api/adoptions", () => ({ adoptionsClient: { listScenarios } }));
vi.mock("../api/components", () => ({ listCatalogAssets }));

import { ActionLauncher } from "./ActionLauncher";
import { renderWithProviders } from "@vrooli/api-base/testing";

function renderLauncher(action: "menu" | "extract" | "adopt" = "menu") {
  return renderWithProviders(
    <ActionLauncher action={action} onActionChange={() => undefined} onCreate={() => undefined} />,
  );
}

describe("ActionLauncher", () => {
  beforeEach(() => {
    startWorkflow.mockResolvedValue({ workflow: { id: "workflow-1" } });
    listScenarios.mockResolvedValue({
      scenarios: ["demo", "demo-a", "demo-b"].map((name) => ({ name, displayName: name })),
    });
    listCatalogAssets.mockResolvedValue({
      components: [{ id: "cmp-1", libraryId: "rcl:Button", displayName: "Button" }],
    });
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("starts extract-assist through the workflow dispatcher", async () => {
    const user = userEvent.setup();
    renderLauncher("extract");
    await screen.findByRole("option", { name: "demo" });
    await user.selectOptions(screen.getByLabelText("catalog.sourceScenario"), "demo");
    await user.type(screen.getByLabelText("catalog.sourcePath"), "ui/src/Panel.tsx");
    await user.click(screen.getByRole("button", { name: "launcher.startExtract" }));
    await waitFor(() =>
      expect(startWorkflow).toHaveBeenCalledWith(
        expect.objectContaining({
          kind: 1,
          sourceScenario: "demo",
          sourcePath: "ui/src/Panel.tsx",
        }),
      ),
    );
  });

  it("starts one adopt-assist workflow for each target", async () => {
    const user = userEvent.setup();
    renderLauncher("adopt");
    await screen.findByRole("option", { name: "Button" });
    await user.selectOptions(screen.getByLabelText("launcher.asset"), "cmp-1");
    await user.selectOptions(screen.getByLabelText("launcher.targets"), ["demo-a", "demo-b"]);
    await user.click(screen.getByRole("button", { name: "launcher.startAdopt" }));
    await waitFor(() => expect(startWorkflow.mock.calls).toHaveLength(2));
    expect(startWorkflow).toHaveBeenCalledWith(
      expect.objectContaining({ assetId: "cmp-1", targetScenario: "demo-a" }),
    );
    expect(startWorkflow).toHaveBeenCalledWith(
      expect.objectContaining({ assetId: "cmp-1", targetScenario: "demo-b" }),
    );
  });

  it("filters typed picker options before selection", async () => {
    const user = userEvent.setup();
    renderLauncher("extract");
    await screen.findByRole("option", { name: "demo-a" });
    await user.type(screen.getByRole("searchbox"), "demo-a");
    expect(screen.getByRole("option", { name: "demo-a" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "demo-b" })).not.toBeInTheDocument();
  });

  it("routes each menu action through its URL action state", async () => {
    const user = userEvent.setup();
    const onActionChange = vi.fn();
    const onCreate = vi.fn();
    renderWithProviders(
      <ActionLauncher action="menu" onActionChange={onActionChange} onCreate={onCreate} />,
    );

    await user.click(screen.getByTestId("launcher-extract"));
    await user.click(screen.getByTestId("launcher-adopt"));
    await user.click(screen.getByTestId("launcher-create"));

    expect(onActionChange).toHaveBeenCalledWith("extract");
    expect(onActionChange).toHaveBeenCalledWith("adopt");
    expect(onCreate).toHaveBeenCalledTimes(1);
  });

  it("shows a dispatch failure without leaving the overlay", async () => {
    startWorkflow.mockRejectedValueOnce(new Error("offline"));
    const user = userEvent.setup();
    renderLauncher("extract");
    await screen.findByRole("option", { name: "demo" });
    await user.selectOptions(screen.getByLabelText("catalog.sourceScenario"), "demo");
    await user.type(screen.getByLabelText("catalog.sourcePath"), "ui/Button.tsx");
    await user.click(screen.getByRole("button", { name: "launcher.startExtract" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("launcher.startError");
  });

  it("keeps an adopt workflow error in the guided picker surface", async () => {
    startWorkflow.mockRejectedValueOnce(new Error("offline"));
    const user = userEvent.setup();
    renderLauncher("adopt");
    await screen.findByRole("option", { name: "Button" });
    await user.selectOptions(screen.getByLabelText("launcher.asset"), "cmp-1");
    await user.selectOptions(screen.getByLabelText("launcher.targets"), "demo-a");
    await user.click(screen.getByRole("button", { name: "launcher.startAdopt" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("launcher.startError");
  });
});
