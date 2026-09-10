import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import { describe, expect, it, vi } from "vitest";
import { DevelopmentResponseSchema } from "@vrooli/proto-types/swarm-manager/v1/api/development_pb";
import { PreviewDevelopmentRequestSchema, PreviewDevelopmentResponseSchema } from "@vrooli/proto-types/swarm-manager/v1/api/transition_pb";
import { DevelopmentContractPanel } from "./development-contract-panel";
import type { DevelopmentClient } from "../../services/development-service";

const item = "execute/example-development";
const proposal = JSON.stringify({ scenario: "example", work_item: item, objective: "Preserve final tail", max_tokens: "10000", max_wall_seconds: "600" });

function fakeClient() {
  return {
    getDevelopment: vi.fn().mockRejectedValue(new ConnectError("No retained contract", Code.NotFound)),
    previewDevelopment: vi.fn().mockResolvedValue(create(PreviewDevelopmentResponseSchema, { proposalDigest: "reviewed-sha", reviewComplete: true, goalMessage: "DRAFT — preserve captured tail", launchBlockers: ["Runtime grants are not qualified"] })),
    approveDevelopment: vi.fn().mockResolvedValue(create(DevelopmentResponseSchema, { workItem: item, workShape: "contract-development", version: 1n, digest: "reviewed-sha", status: "approved", launchBlockers: ["Runtime grants are not qualified"] })),
    revokeDevelopment: vi.fn(),
    acceptDevelopment: vi.fn(),
    getDevelopmentArtifact: vi.fn(),
  };
}

async function openPanel(client = fakeClient()) {
  renderWithProviders(<DevelopmentContractPanel workItem={item} client={client as DevelopmentClient} />);
  fireEvent.click(screen.getByRole("button", { name: /Adaptive plan/ }));
  await screen.findByRole("textbox", { name: "Proposal JSON" });
  return client;
}

describe("development contract review [REQ:SWM-P0-017]", () => {
  it("requires amendment acknowledgement and retains the owner's version", async () => {
    const client = fakeClient();
    client.getDevelopment.mockResolvedValue(create(DevelopmentResponseSchema, {
      workItem: item, workShape: "contract-development", version: 7n, digest: "retained", status: "approved",
      approvedProposal: create(PreviewDevelopmentRequestSchema, { workItem: item, scenario: "example", objective: "Existing outcome", maxTokens: 1000n }),
    }));
    await openPanel(client);
    fireEvent.change(screen.getByRole("textbox", { name: "Proposal JSON" }), { target: { value: proposal } });
    fireEvent.click(screen.getByRole("button", { name: "Preview plan strategy and goal message" }));
    await screen.findByText("DRAFT — preserve captured tail");
    fireEvent.change(screen.getByRole("textbox", { name: "Decision rationale" }), { target: { value: "Reviewed expanded allowance" } });
    const approve = screen.getByRole("button", { name: "Approve reviewed amendment (no launch)" });
    expect(approve).toBeDisabled();
    expect(screen.getByText("max_tokens")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("checkbox", { name: "I reviewed these changes and the new goal message" }));
    expect(approve).toBeEnabled();
    fireEvent.click(approve);
    await waitFor(() => expect(client.approveDevelopment).toHaveBeenCalledWith(expect.objectContaining({ expectedVersion: 7n, reviewedDigest: "reviewed-sha" })));
  });
  it("does not fetch or approve until the operator opens and reviews", async () => {
    const client = fakeClient();
    renderWithProviders(<DevelopmentContractPanel workItem={item} client={client as DevelopmentClient} />);
    expect(client.getDevelopment).not.toHaveBeenCalled();
    expect(client.approveDevelopment).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: /launch agent/i })).not.toBeInTheDocument();
  });

  it("requires a fresh preview and rationale; edits invalidate review", async () => {
    const client = await openPanel();
    const approve = screen.getByRole("button", { name: "Approve reviewed target (no launch)" });
    expect(approve).toBeDisabled();
    fireEvent.change(screen.getByRole("textbox", { name: "Proposal JSON" }), { target: { value: proposal } });
    fireEvent.click(screen.getByRole("button", { name: "Preview plan strategy and goal message" }));
    await screen.findByText("DRAFT — preserve captured tail");
    expect(approve).toBeDisabled();
    fireEvent.change(screen.getByRole("textbox", { name: "Decision rationale" }), { target: { value: "Reviewed local-only target" } });
    expect(approve).toBeEnabled();
    fireEvent.change(screen.getByRole("textbox", { name: "Proposal JSON" }), { target: { value: proposal + " " } });
    expect(approve).toBeDisabled();
    expect(client.approveDevelopment).not.toHaveBeenCalled();
  });

  it("binds approval to the reviewed digest and never calls execution", async () => {
    const client = await openPanel();
    fireEvent.change(screen.getByRole("textbox", { name: "Proposal JSON" }), { target: { value: proposal } });
    fireEvent.click(screen.getByRole("button", { name: "Preview plan strategy and goal message" }));
    await screen.findByText("DRAFT — preserve captured tail");
    fireEvent.change(screen.getByRole("textbox", { name: "Decision rationale" }), { target: { value: "Approved target only" } });
    fireEvent.click(screen.getByRole("button", { name: "Approve reviewed target (no launch)" }));
    await waitFor(() => expect(client.approveDevelopment).toHaveBeenCalledWith(expect.objectContaining({ reviewedDigest: "reviewed-sha", expectedVersion: 0n, reason: "Approved target only" })));
    await screen.findByText(/Strategy: adaptive improvement/);
    expect(screen.getByText("Runtime grants are not qualified")).toBeInTheDocument();
    expect(client.acceptDevelopment).not.toHaveBeenCalled();
  });

  it("does not send another item's proposal or mistake an outage for absence", async () => {
    const client = await openPanel();
    fireEvent.change(screen.getByRole("textbox", { name: "Proposal JSON" }), { target: { value: proposal.replace(item, "execute/another-item") } });
    fireEvent.click(screen.getByRole("button", { name: "Preview plan strategy and goal message" }));
    await screen.findByRole("alert");
    expect(client.previewDevelopment).not.toHaveBeenCalled();
    client.getDevelopment.mockRejectedValue(new ConnectError("database unavailable", Code.Unavailable));
    fireEvent.click(screen.getByRole("button", { name: "Reload retained state" }));
    await screen.findByText(/database unavailable/);
    expect(screen.queryByRole("button", { name: "Approve reviewed target (no launch)" })).not.toBeInTheDocument();
  });

  it("shows a stale approval error without replacing retained authority", async () => {
    const client = await openPanel();
    client.approveDevelopment.mockRejectedValue(new ConnectError("target changed; reload", Code.Aborted));
    fireEvent.change(screen.getByRole("textbox", { name: "Proposal JSON" }), { target: { value: proposal } });
    fireEvent.click(screen.getByRole("button", { name: "Preview plan strategy and goal message" }));
    await screen.findByText("DRAFT — preserve captured tail");
    fireEvent.change(screen.getByRole("textbox", { name: "Decision rationale" }), { target: { value: "Review complete" } });
    fireEvent.click(screen.getByRole("button", { name: "Approve reviewed target (no launch)" }));
    await screen.findByText(/target changed; reload/);
    expect(screen.queryByText(/Status: approved/)).not.toBeInTheDocument();
  });

  it("shows checkpoint progress and keeps remaining outcomes visible", async () => {
    const client = fakeClient();
    client.getDevelopment.mockResolvedValue(create(DevelopmentResponseSchema, {
      workItem: item, workShape: "contract-development", version: 2n, digest: "retained", status: "paused",
      campaign: { approvalDigest: "retained", attemptKey: "repair-two", ownerExecutionId: "owner-two", pending: false, remainingOutcomeIds: ["streaming"], noProgressCycles: 1, lastCheckpoint: { kind: "owner-verification", value: "measured" } },
    }));
    await openPanel(client);
    expect(screen.getByText(/Remaining outcomes: streaming/)).toBeInTheDocument();
    expect(screen.getByText(/No useful progress observed/)).toBeInTheDocument();
  });
});
