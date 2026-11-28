import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RehearsalState, ReviewDecision } from "@vrooli/proto-types/client-rehearsal/v1/rehearsals/rehearsals_pb";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { makeRehearsal } from "./mocks/factories";

const mocks = vi.hoisted(() => ({
  client: {
    listRehearsals: vi.fn(),
    getRehearsal: vi.fn(),
    createAndRunRehearsal: vi.fn(),
    recordReview: vi.fn(),
  },
}));

vi.mock("../../api/rehearsals", () => ({ rehearsalClient: mocks.client }));

import { RehearsalWorkspace } from "./RehearsalWorkspace";

function workflowFile(content: string, name = "workflow.json") {
  const file = new File([content], name, { type: "application/json" });
  // jsdom's File implementation does not currently expose Blob.text().
  Object.defineProperty(file, "text", { value: async () => content });
  return file;
}

afterEach(() => cleanup());
beforeEach(() => {
  vi.clearAllMocks();
  mocks.client.listRehearsals.mockResolvedValue({ rehearsals: [] });
  mocks.client.getRehearsal.mockImplementation(async () => ({ rehearsal: makeRehearsal() }));
  mocks.client.createAndRunRehearsal.mockResolvedValue({ rehearsal: makeRehearsal() });
  mocks.client.recordReview.mockResolvedValue({ rehearsal: makeRehearsal({ state: RehearsalState.APPROVED }) });
});

describe("RehearsalWorkspace", () => {
  it("imports and runs an exported workflow using the chosen trigger and synthetic simulator", async () => {
    const user = userEvent.setup();
    renderWithProviders(<RehearsalWorkspace />);
    const workflow = {
      nodes: [
        { id: "manual-1", name: "Manual start", type: "n8n-nodes-base.manualTrigger" },
        { id: "webhook-1", name: "Lead webhook", type: "n8n-nodes-base.webhook" },
        { id: "error-1", name: "Error handler", type: "n8n-nodes-base.errorTrigger" },
      ],
    };
    const file = workflowFile(JSON.stringify(workflow), "leads.json");
    await user.upload(screen.getByTestId(selectors.rehearsals.fileInput), file);
    await waitFor(() => expect(screen.getByTestId(selectors.rehearsals.triggerSelect)).toBeInTheDocument());
    await user.selectOptions(screen.getByTestId(selectors.rehearsals.triggerSelect), "webhook-1");
    await user.type(screen.getByLabelText(strings.rehearsals.titleLabel), " CRM handoff ");
    await user.click(screen.getByTestId(selectors.rehearsals.runButton));

    await waitFor(() => expect(mocks.client.createAndRunRehearsal).toHaveBeenCalledOnce());
    const request = mocks.client.createAndRunRehearsal.mock.calls[0]?.[0];
    expect(request).toBeDefined();
    expect(request).toMatchObject({
      title: "CRM handoff",
      workflowJson: JSON.stringify(workflow),
      triggerNodeId: "webhook-1",
      simulatorProfile: "synthetic-default",
      injectSeededFault: true,
    });
    expect(JSON.parse(request.simulatorSourceJson).services.map((service: { id: string }) => service.id)).toEqual(["crm", "sheets"]);
    expect(await screen.findByTestId(selectors.rehearsals.detail)).toHaveTextContent("Synthetic CRM contact was upserted");
    expect(screen.getByTestId(selectors.rehearsals.trace)).toHaveTextContent("crm.upsertContact");
  });

  it("requests a server-owned Workspace Sandbox task scope when selected", async () => {
    const user = userEvent.setup();
    renderWithProviders(<RehearsalWorkspace />);
    await user.upload(screen.getByTestId(selectors.rehearsals.fileInput), workflowFile(JSON.stringify({ nodes: [{ id: "start", type: "n8n-nodes-base.manualTrigger" }] })));
    await user.click(screen.getByLabelText(strings.rehearsals.taskScopeLabel));
    await user.click(screen.getByTestId(selectors.rehearsals.runButton));

    await waitFor(() => expect(mocks.client.createAndRunRehearsal).toHaveBeenCalledOnce());
    expect(mocks.client.createAndRunRehearsal.mock.calls[0]?.[0]).toMatchObject({ taskScoped: true });
  });

  it("rejects malformed simulator input before sending workflow data to the API", async () => {
    const user = userEvent.setup();
    renderWithProviders(<RehearsalWorkspace />);
    const file = workflowFile(JSON.stringify({ nodes: [{ id: "start", type: "n8n-nodes-base.manualTrigger" }] }));
    await user.upload(screen.getByTestId(selectors.rehearsals.fileInput), file);
    await user.click(screen.getByText(strings.rehearsals.simulatorAdvanced));
    fireEvent.change(screen.getByTestId(selectors.rehearsals.simulatorSource), { target: { value: "{broken json" } });
    await user.click(screen.getByTestId(selectors.rehearsals.runButton));

    expect(await screen.findByRole("alert")).toHaveTextContent(strings.rehearsals.invalidSimulator);
    expect(mocks.client.createAndRunRehearsal).not.toHaveBeenCalled();
  });

  it("loads saved run details and records an explicit review decision", async () => {
    const user = userEvent.setup();
    const saved = makeRehearsal({ state: RehearsalState.NEEDS_REVIEW });
    mocks.client.listRehearsals.mockResolvedValue({ rehearsals: [saved] });
    mocks.client.getRehearsal.mockResolvedValue({ rehearsal: saved });
    renderWithProviders(<RehearsalWorkspace />);
    await user.click(await screen.findByTestId(selectors.rehearsals.rehearsalRow));
    expect(await screen.findByTestId(selectors.rehearsals.detail)).toHaveTextContent("Synthetic CRM contact was upserted");
    expect(screen.getByTestId(selectors.rehearsals.detail)).toHaveTextContent("seeded CRM timeout");
    await user.click(screen.getByTestId(selectors.rehearsals.approveButton));

    await waitFor(() => expect(mocks.client.recordReview).toHaveBeenCalledWith({
      id: saved.id,
      decision: ReviewDecision.APPROVE,
      note: "Reviewed synthetic run evidence in the rehearsal workspace.",
    }));
    await waitFor(() => expect(mocks.client.getRehearsal).toHaveBeenCalled());
  });

  it("surfaces list and run failures as accessible errors", async () => {
    mocks.client.listRehearsals.mockRejectedValueOnce(new Error("history unavailable"));
    renderWithProviders(<RehearsalWorkspace />);
    await waitFor(() => expect(screen.getByTestId(selectors.rehearsals.surface)).toHaveAttribute("data-experience-state", "error"));

    cleanup();
    vi.clearAllMocks();
    mocks.client.listRehearsals.mockResolvedValue({ rehearsals: [] });
    mocks.client.createAndRunRehearsal.mockRejectedValue(new Error("run failed"));
    const user = userEvent.setup();
    renderWithProviders(<RehearsalWorkspace />);
    await user.upload(screen.getByTestId(selectors.rehearsals.fileInput), workflowFile("{}", "empty.json"));
    await user.click(screen.getByTestId(selectors.rehearsals.runButton));
    expect(await screen.findByRole("alert")).toHaveTextContent("run failed");
  });
});
