import { create } from "@bufbuild/protobuf";
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { ListPhotographsResponseSchema, PhotographResponseSchema, StoredPhotographSchema } from "@vrooli/proto-types/image-tools/v1/preparation/preparation_pb";

import { renderWithProviders } from "../../test-utils";
import { preparationClient } from "../../api/preparation";
import { strings } from "../../consts/strings";
import { PreparedPhotographs } from "./PreparedPhotographs";

vi.mock("../../api/preparation", () => ({ preparationClient: { listPhotographs: vi.fn(), reviewPhotograph: vi.fn() } }));
afterEach(() => { cleanup(); vi.clearAllMocks(); });

describe("PreparedPhotographs", () => {
  it("compares retained rasters and records a rejection as a child without approving the source", async () => {
    const pending = create(StoredPhotographSchema, { id: "source-id", label: "Desk", sourcePreviewRef: "scene-preparation-assets/source", cleanPlatePreviewRef: "scene-preparation-assets/clean", manifest: { review: { decision: "pending" } } });
    const rejected = create(StoredPhotographSchema, { label: pending.label, sourcePreviewRef: pending.sourcePreviewRef, cleanPlatePreviewRef: pending.cleanPlatePreviewRef, id: "child-id", parentId: pending.id, manifest: { review: { decision: "rejected", reviewedBy: "Reviewer", rationale: "Old shadow remains" } } });
    vi.mocked(preparationClient.listPhotographs).mockResolvedValueOnce(create(ListPhotographsResponseSchema, { photographs: [pending] })).mockResolvedValue(create(ListPhotographsResponseSchema, { photographs: [rejected, pending] }));
    vi.mocked(preparationClient.reviewPhotograph).mockResolvedValue(create(PhotographResponseSchema, { photograph: rejected }));
    renderWithProviders(<PreparedPhotographs />);
    fireEvent.click(await screen.findByRole("button", { name: `Desk · ${strings.preparation.status.pending}` }));
    expect(screen.getByAltText(strings.preparation.sourceAlt)).toHaveAttribute("src", expect.stringContaining("scene-preparation-assets/source"));
    expect(screen.getByAltText(strings.preparation.cleanPlateAlt)).toBeVisible();
    expect(screen.getByRole("button", { name: strings.preparation.accept })).toBeDisabled();
    fireEvent.change(screen.getByLabelText(strings.preparation.reviewer), { target: { value: "Reviewer" } });
    fireEvent.change(screen.getByLabelText(strings.preparation.rationale), { target: { value: "Old shadow remains" } });
    fireEvent.change(screen.getByLabelText(strings.preparation.evidence), { target: { value: "sample/frame.png" } });
    fireEvent.click(screen.getByRole("button", { name: strings.preparation.reject }));
    await waitFor(() => expect(preparationClient.reviewPhotograph).toHaveBeenCalledWith({ id: "source-id", review: { decision: "rejected", reviewedBy: "Reviewer", rationale: "Old shadow remains", evidenceRefs: ["sample/frame.png"] }, overscanReviewed: false, operationId: expect.stringMatching(/^[0-9a-f-]{36}$/) }));
    expect(await screen.findByRole("button", { name: `Desk · ${strings.preparation.status.rejected}` })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: `Desk · ${strings.preparation.status.pending}` })).toHaveAttribute("aria-pressed", "false");
  });

  it("keeps a failed review visible with correction inputs retained", async () => {
    vi.mocked(preparationClient.listPhotographs).mockResolvedValue(create(ListPhotographsResponseSchema, { photographs: [create(StoredPhotographSchema, { id: "id", label: "Desk", manifest: { review: { decision: "pending" } } })] }));
    vi.mocked(preparationClient.reviewPhotograph).mockRejectedValue(new Error("Review unavailable"));
    renderWithProviders(<PreparedPhotographs />);
    fireEvent.click(await screen.findByRole("button", { name: `Desk · ${strings.preparation.status.pending}` }));
    for (const [label,value] of [[strings.preparation.reviewer,"Reviewer"],[strings.preparation.rationale,"visible texture mismatch"],[strings.preparation.evidence,"frame.png"]] as const) { fireEvent.change(screen.getByLabelText(label),{ target:{ value } }); }
    fireEvent.click(screen.getByRole("button", { name: strings.preparation.reject }));
    expect(await screen.findByRole("alert")).toBeVisible();
    expect(screen.getByLabelText(strings.preparation.rationale)).toHaveValue("visible texture mismatch");
    expect(screen.getByRole("button", { name: strings.preparation.reject })).toBeEnabled();
  });

  it("reuses the exact operation ID when the same review is retried after an uncertain failure", async () => {
    vi.mocked(preparationClient.listPhotographs).mockResolvedValue(create(ListPhotographsResponseSchema, { photographs: [create(StoredPhotographSchema, { id: "id", label: "Desk", manifest: { review: { decision: "pending" } } })] }));
    vi.mocked(preparationClient.reviewPhotograph).mockRejectedValueOnce(new Error("response lost")).mockResolvedValue(create(PhotographResponseSchema, { photograph: create(StoredPhotographSchema, { id: "child", parentId: "id", manifest: { review: { decision: "rejected" } } }) }));
    renderWithProviders(<PreparedPhotographs />);
    fireEvent.click(await screen.findByRole("button", { name: `Desk · ${strings.preparation.status.pending}` }));
    for (const [label, value] of [[strings.preparation.reviewer, "Reviewer"], [strings.preparation.rationale, "Visible mismatch"], [strings.preparation.evidence, "frame.png"]] as const) fireEvent.change(screen.getByLabelText(label), { target: { value } });
    fireEvent.click(screen.getByRole("button", { name: strings.preparation.reject }));
    await screen.findByRole("alert");
    const first = vi.mocked(preparationClient.reviewPhotograph).mock.calls[0]?.[0];
    expect(first).toBeDefined();
    fireEvent.click(screen.getByRole("button", { name: strings.preparation.reject }));
    await waitFor(() => expect(preparationClient.reviewPhotograph).toHaveBeenCalledTimes(2));
    expect(vi.mocked(preparationClient.reviewPhotograph).mock.calls[1]?.[0]).toEqual(first);
  });
});
