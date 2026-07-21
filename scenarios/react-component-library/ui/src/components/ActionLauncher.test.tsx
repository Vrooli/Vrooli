import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const { startWorkflow } = vi.hoisted(() => ({ startWorkflow: vi.fn() }));

vi.mock("../api/workflows", () => ({ workflowsClient: { startWorkflow } }));

import { ActionLauncher } from "./ActionLauncher";

function renderLauncher(action: "menu" | "extract" | "adopt" = "menu") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={queryClient}><ActionLauncher action={action} onActionChange={() => undefined} onCreate={() => undefined} /></QueryClientProvider>);
}

describe("ActionLauncher", () => {
  beforeEach(() => startWorkflow.mockResolvedValue({ workflow: { id: "workflow-1" } }));
  afterEach(() => { cleanup(); vi.clearAllMocks(); });

  it("starts extract-assist through the workflow dispatcher", async () => {
    const user = userEvent.setup();
    renderLauncher("extract");
    await user.type(screen.getByLabelText("catalog.sourceScenario"), "demo");
    await user.type(screen.getByLabelText("catalog.sourcePath"), "ui/src/Panel.tsx");
    await user.click(screen.getByRole("button", { name: "launcher.startExtract" }));
    await waitFor(() => expect(startWorkflow).toHaveBeenCalledWith(expect.objectContaining({ kind: 1, sourceScenario: "demo", sourcePath: "ui/src/Panel.tsx" })));
  });

  it("starts one adopt-assist workflow for each target", async () => {
    const user = userEvent.setup();
    renderLauncher("adopt");
    await user.type(screen.getByLabelText("launcher.asset"), "cmp-1");
    await user.type(screen.getByLabelText("launcher.targets"), "demo-a, demo-b");
    await user.click(screen.getByRole("button", { name: "launcher.startAdopt" }));
    await waitFor(() => expect(startWorkflow.mock.calls.filter(([input]) => input?.kind === 2)).toHaveLength(2));
    expect(startWorkflow).toHaveBeenCalledWith(expect.objectContaining({ kind: 2, assetId: "cmp-1", targetScenario: "demo-a" }));
    expect(startWorkflow).toHaveBeenCalledWith(expect.objectContaining({ kind: 2, assetId: "cmp-1", targetScenario: "demo-b" }));
  });
});
