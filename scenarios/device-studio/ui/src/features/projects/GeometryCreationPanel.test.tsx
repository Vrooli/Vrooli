import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { MediaReadiness, SourceMediaKind, SourceMediaSchema } from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { cancelGeometryCreation, cancelGeometryCreationSession, createGeometryCreationSession, findGeometryCreationOperation, getGeometryCorrection, getGeometryCreation, getGeometryCreationSession, refineGeometryCreationSession, startGeometryCreation } from "../../api/creation";
import { projectsClient } from "../../api/projects";
import { getLocalGeometryCreationLineage } from "../../api/library-intake";
import { GeometryCreationPanel } from "./GeometryCreationPanel";

vi.mock("../../api/projects", () => ({ projectsClient: { listSourceMedia: vi.fn() } }));
vi.mock("../../api/library-intake", () => ({ getLocalGeometryCreationLineage: vi.fn() }));
vi.mock("../../api/creation", () => ({ startGeometryCreation: vi.fn(), createGeometryCreationSession: vi.fn(), getGeometryCreationSession: vi.fn(), refineGeometryCreationSession: vi.fn(), cancelGeometryCreationSession: vi.fn(), findGeometryCreationOperation: vi.fn(), getGeometryCorrection: vi.fn(), getGeometryCreation: vi.fn(), cancelGeometryCreation: vi.fn() }));

const media = create(SourceMediaSchema, { id: "image-1", projectId: "p1", originalFilename: "device.png", mediaType: "image/png", sha256: "a".repeat(64), sizeBytes: 1024n, kind: SourceMediaKind.IMAGE, readiness: MediaReadiness.READY, width: 800, height: 1200 });

describe("GeometryCreationPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
    localStorage.clear();
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [media], nextPageToken: "" } as never);
    vi.mocked(findGeometryCreationOperation).mockResolvedValue(null);
    vi.mocked(startGeometryCreation).mockResolvedValue({ job_id: "job-1", project_id: "p1", candidate_id: "creation-1", state: "queued", version: 1, review_status: "unreviewed" });
    const session = { session_id: "session-1", state: "active", version: 2, policy: { project_id: "p1", candidate_id: "creation-1", brief: "Brief", envelope_mm: [78, 148, 12], reference_media_ids: ["image-1"], funding_route: "local", tokens_per_call: 4096 }, max_provider_calls: 2, provider_calls_reserved: 1, max_children: 2, children_admitted: 1, children: [{ job_id: "job-1", ordinal: 1, state: "queued", version: 1, candidate_id: "creation-1" }] } as const;
    vi.mocked(createGeometryCreationSession).mockResolvedValue(session as never);
    vi.mocked(getGeometryCreationSession).mockResolvedValue(session as never);
    vi.mocked(refineGeometryCreationSession).mockResolvedValue(session as never);
    vi.mocked(cancelGeometryCreationSession).mockResolvedValue({ ...session, state: "cancelled", version: 4 } as never);
    vi.mocked(getGeometryCreation).mockResolvedValue({ job_id: "job-1", project_id: "p1", candidate_id: "creation-1", state: "queued", version: 1, review_status: "unreviewed" });
    vi.mocked(getGeometryCorrection).mockResolvedValue({ job_id: "job-1", project_id: "p1", disposition: "unavailable", envelope_mm: [78, 148, 12] });
    vi.mocked(cancelGeometryCreation).mockResolvedValue({ job_id: "job-1", project_id: "p1", candidate_id: "creation-1", state: "cancel_requested", version: 2, review_status: "unreviewed" });
    vi.mocked(getLocalGeometryCreationLineage).mockResolvedValue(undefined);
  });
  afterEach(() => { cleanup(); vi.clearAllMocks(); });

  it("requires clear local-AI consent, bounded dimensions and selected project image before dispatch", async () => {
    const user = userEvent.setup();
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    const createButton = await screen.findByRole("button", { name: "Create geometry draft" });
    expect(createButton).toBeDisabled();
    await user.type(screen.getByLabelText("Describe the device"), "A compact field phone with physical controls");
    await user.type(screen.getByLabelText("Width (mm)"), "78");
    await user.type(screen.getByLabelText("Height (mm)"), "148");
    await user.type(screen.getByLabelText("Depth (mm)"), "12");
    await user.click(screen.getByLabelText("device.png · image/png · 1024 bytes"));
    expect(createButton).toBeDisabled();
    await user.click(screen.getByLabelText("I confirm this local AI request."));
    expect(createButton).toBeEnabled();
    await user.click(createButton);
    await waitFor(() => expect(createGeometryCreationSession).toHaveBeenCalledWith("p1", expect.objectContaining({ brief: "A compact field phone with physical controls", envelopeMM: [78, 148, 12], referenceMediaIDs: ["image-1"], fundingRoute: "local" })));
    expect(screen.getByText(/at most two provider calls, each capped at 4096 output tokens/)).toBeInTheDocument();
    expect(await screen.findByTestId("geometry-creation-status")).toHaveTextContent("unreviewed");
  });

  it("requires a separate confirmation for a selected remote funding route", async () => {
    const user = userEvent.setup();
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    await user.type(await screen.findByLabelText("Describe the device"), "Compact field device");
    await user.type(screen.getByLabelText("Width (mm)"), "78");
    await user.type(screen.getByLabelText("Height (mm)"), "148");
    await user.type(screen.getByLabelText("Depth (mm)"), "12");
    await user.click(screen.getByLabelText("device.png · image/png · 1024 bytes"));
    await user.selectOptions(screen.getByLabelText("AI funding route"), "byok");
    expect(screen.getByText(/will be sent to OpenRouter BYOK/)).toBeInTheDocument();
    const button = screen.getByRole("button", { name: "Create geometry draft" });
    expect(button).toBeDisabled();
    await user.type(screen.getByLabelText(/Maximum spend/), "0.25");
    expect(screen.getByText(/Price estimate: unknown/)).toBeInTheDocument();
    expect(button).toBeDisabled();
    await user.click(screen.getByLabelText(/I confirm sending these project images/));
    expect(button).toBeEnabled();
    await user.click(button);
    await waitFor(() => expect(startGeometryCreation).toHaveBeenCalledWith("p1", expect.objectContaining({ fundingRoute: "byok", maxCostMicros: 250_000 })));
  });

  it("shows settled remote USD and credits from the exact candidate lineage, preserving zero credits", async () => {
    const request = { candidateId: "creation-funded", idempotencyKey: "request-funded", brief: "Compact field device", envelopeMM: [78, 148, 12] as [number, number, number], referenceMediaIDs: ["image-1"], fundingRoute: "vrooli_credits" as const, maxCostMicros: 100_000 };
    localStorage.setItem("device-studio:geometry-creation:p1", JSON.stringify({ request, jobId: "job-funded", phase: "pending" }));
    vi.mocked(getGeometryCreation).mockResolvedValueOnce({ job_id: "job-funded", project_id: "p1", candidate_id: "creation-funded", state: "succeeded", version: 3, review_status: "unreviewed" });
    vi.mocked(getLocalGeometryCreationLineage).mockResolvedValueOnce({ costMicros: 10_000n, costProvenance: 2, creditsCharged: 0n, accountingStatus: "settled" } as never);
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    expect(await screen.findByText("Actual provider cost: 0.01 USD; 0 Vrooli credits charged.")).toBeInTheDocument();
    expect(getLocalGeometryCreationLineage).toHaveBeenCalledWith("creation-funded", 1);
  });

  it("does not enable hosted fallback or call the job owner when local images are unavailable", async () => {
    vi.mocked(projectsClient.listSourceMedia).mockRejectedValueOnce(new Error("offline"));
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Project images could not be loaded.");
    expect(startGeometryCreation).not.toHaveBeenCalled();
    expect(screen.getByText(/there is no hosted fallback/i)).toBeInTheDocument();
  });

  it("recovers an uncertain submit after remount and retries with the same durable identity", async () => {
    const user = userEvent.setup();
    vi.mocked(createGeometryCreationSession).mockRejectedValueOnce(new Error("connection dropped"));
    const first = renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    await user.type(await screen.findByLabelText("Describe the device"), "Compact field device");
    await user.type(screen.getByLabelText("Width (mm)"), "78");
    await user.type(screen.getByLabelText("Height (mm)"), "148");
    await user.type(screen.getByLabelText("Depth (mm)"), "12");
    await user.click(screen.getByLabelText("device.png · image/png · 1024 bytes"));
    await user.click(screen.getByLabelText("I confirm this local AI request."));
    await user.click(screen.getByRole("button", { name: "Create geometry draft" }));
    await screen.findByRole("alert");
    const firstRequest = vi.mocked(createGeometryCreationSession).mock.calls[0]?.[1];
    expect(firstRequest).toBeDefined();
    first.unmount();

    vi.mocked(createGeometryCreationSession).mockResolvedValueOnce({ session_id: "session-recovered", state: "active", version: 2, policy: { project_id: "p1", candidate_id: firstRequest!.candidateId, brief: firstRequest!.brief, envelope_mm: firstRequest!.envelopeMM, reference_media_ids: firstRequest!.referenceMediaIDs, funding_route: "local", tokens_per_call: 4096 }, max_provider_calls: 2, provider_calls_reserved: 1, max_children: 2, children_admitted: 1, children: [{ job_id: "job-recovered", ordinal: 1, state: "queued", version: 1, candidate_id: firstRequest!.candidateId }] } as never);
    vi.mocked(getGeometryCreation).mockResolvedValueOnce({ job_id: "job-recovered", project_id: "p1", candidate_id: firstRequest!.candidateId, state: "queued", version: 1, review_status: "unreviewed" });
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    const retry = await screen.findByRole("button", { name: "Retry the same request" });
    await user.click(retry);
    await waitFor(() => expect(createGeometryCreationSession).toHaveBeenCalledTimes(2));
    expect(vi.mocked(createGeometryCreationSession).mock.calls[1]).toEqual(["p1", firstRequest]);
    expect(findGeometryCreationOperation).toHaveBeenCalledWith("p1", firstRequest!.idempotencyKey);
    expect(await screen.findByTestId("geometry-creation-status")).toHaveTextContent("job-recovered");
  });

  it("restores owner-confirmed status after reload without starting another request", async () => {
    const request = { candidateId: "creation-stable", idempotencyKey: "request-stable", brief: "Compact field device", envelopeMM: [78, 148, 12] as [number, number, number], referenceMediaIDs: ["image-1"], fundingRoute: "local" };
    localStorage.setItem("device-studio:geometry-creation:p1", JSON.stringify({ request }));
    vi.mocked(findGeometryCreationOperation).mockResolvedValueOnce({ job_id: "job-owner", project_id: "p1", candidate_id: request.candidateId, state: "running", version: 2, review_status: "unreviewed" });
    vi.mocked(getGeometryCreation).mockResolvedValueOnce({ job_id: "job-owner", project_id: "p1", candidate_id: request.candidateId, state: "running", version: 2, review_status: "unreviewed" });
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    expect(await screen.findByTestId("geometry-creation-status")).toHaveTextContent("job-owner");
    expect(findGeometryCreationOperation).toHaveBeenCalledWith("p1", request.idempotencyKey);
    expect(startGeometryCreation).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: "Create geometry draft" })).not.toBeInTheDocument();
  });

  it("does not submit when durable operation storage throws, including after remount", async () => {
    const user = userEvent.setup();
    const setItem = vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("storage blocked"); });
    const first = renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    await user.type(await screen.findByLabelText("Describe the device"), "Compact field device");
    await user.type(screen.getByLabelText("Width (mm)"), "78");
    await user.type(screen.getByLabelText("Height (mm)"), "148");
    await user.type(screen.getByLabelText("Depth (mm)"), "12");
    await user.click(screen.getByLabelText("device.png · image/png · 1024 bytes"));
    await user.click(screen.getByLabelText("I confirm this local AI request."));
    await user.click(screen.getByRole("button", { name: "Create geometry draft" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Browser storage is unavailable");
    expect(startGeometryCreation).not.toHaveBeenCalled();
    first.unmount();

    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    expect(await screen.findByRole("button", { name: "Create geometry draft" })).toBeDisabled();
    expect(startGeometryCreation).not.toHaveBeenCalled();
    setItem.mockRestore();
  });

  it("routes cancellation through the job owner for a recovered active operation", async () => {
    const user = userEvent.setup();
    const request = { candidateId: "creation-stable", idempotencyKey: "request-stable", brief: "Compact field device", envelopeMM: [78, 148, 12] as [number, number, number], referenceMediaIDs: ["image-1"], fundingRoute: "local" };
    localStorage.setItem("device-studio:geometry-creation:p1", JSON.stringify({ request, jobId: "job-1" }));
    vi.mocked(getGeometryCreation).mockResolvedValueOnce({ job_id: "job-1", project_id: "p1", candidate_id: request.candidateId, state: "running", version: 3, review_status: "unreviewed" });
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    await user.click(await screen.findByRole("button", { name: "Request cancellation" }));
    await waitFor(() => expect(cancelGeometryCreation).toHaveBeenCalledWith(expect.objectContaining({ job_id: "job-1", version: 3 })));
    expect(await screen.findByTestId("geometry-creation-status")).toHaveTextContent("cancel_requested");
  });

  it("cancels a saved local session by its durable owner version", async () => {
    const user = userEvent.setup();
    const request = { candidateId: "creation-stable", idempotencyKey: "request-stable", brief: "Compact field device", envelopeMM: [78, 148, 12] as [number, number, number], referenceMediaIDs: ["image-1"], fundingRoute: "local" as const };
    localStorage.setItem("device-studio:geometry-creation:p1", JSON.stringify({ request, jobId: "job-1", sessionId: "session-1", phase: "pending" }));
    vi.mocked(getGeometryCreation).mockResolvedValueOnce({ job_id: "job-1", project_id: "p1", candidate_id: request.candidateId, state: "running", version: 3, review_status: "unreviewed" });
    vi.mocked(getGeometryCreationSession).mockResolvedValue({ session_id: "session-1", state: "active", version: 7, policy: { project_id: "p1", candidate_id: request.candidateId, brief: request.brief, envelope_mm: request.envelopeMM, reference_media_ids: request.referenceMediaIDs, funding_route: "local", tokens_per_call: 4096 }, max_provider_calls: 2, provider_calls_reserved: 1, max_children: 2, children_admitted: 1, children: [{ job_id: "job-1", ordinal: 1, state: "running", version: 3, candidate_id: request.candidateId }] } as never);
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    await user.click(await screen.findByRole("button", { name: "Request cancellation" }));
    await waitFor(() => expect(cancelGeometryCreationSession).toHaveBeenCalledWith("session-1", 7));
    expect(cancelGeometryCreation).not.toHaveBeenCalled();
  });

  it("reuses session confirmation only after owner readback proves the correction stays in scope", async () => {
    const user = userEvent.setup();
    const request = { candidateId: "creation-child", idempotencyKey: "refine-key", brief: "Compact field device", envelopeMM: [78, 148, 12] as [number, number, number], referenceMediaIDs: ["image-1"], fundingRoute: "local" as const, correctionSourceJobId: "job-base", correctionMode: "visual_refinement" as const, correctionRecipeSha256: "b".repeat(64), correctionFeedback: "Narrow the bezel." };
    localStorage.setItem("device-studio:geometry-creation:p1", JSON.stringify({ request, sessionId: "session-1", phase: "draft" }));
    vi.mocked(getGeometryCreationSession).mockResolvedValue({ session_id: "session-1", state: "active", version: 2, policy: { project_id: "p1", candidate_id: "creation-base", brief: request.brief, envelope_mm: request.envelopeMM, reference_media_ids: request.referenceMediaIDs, funding_route: "local", tokens_per_call: 4096 }, max_provider_calls: 2, provider_calls_reserved: 1, max_children: 2, children_admitted: 1, children: [{ job_id: "job-base", ordinal: 1, state: "succeeded", version: 4, candidate_id: "creation-base" }] } as never);
    vi.mocked(getGeometryCorrection).mockResolvedValue({ job_id: "job-base", project_id: "p1", disposition: "available", mode: "visual_refinement", envelope_mm: request.envelopeMM, reference_media_ids: request.referenceMediaIDs, prior_recipe_json: "{}", recipe_sha256: request.correctionRecipeSha256, recipe_bytes: 2 });
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    expect(await screen.findByText(/Your earlier confirmation covers this next local call/)).toBeInTheDocument();
    expect(screen.queryByLabelText("I confirm this local AI request.")).not.toBeInTheDocument();
    await user.click(await screen.findByRole("button", { name: "Submit corrected draft" }));
    await waitFor(() => expect(refineGeometryCreationSession).toHaveBeenCalledWith("session-1", expect.objectContaining({ correctionSourceJobId: "job-base", confirmScopeExpansion: false })));
  });

  it("requires an explicit second confirmation before creating a linked changed-scope session", async () => {
    const user = userEvent.setup();
    const request = { candidateId: "creation-child", idempotencyKey: "refine-key", brief: "Changed brief", envelopeMM: [78, 148, 12] as [number, number, number], referenceMediaIDs: ["image-1"], fundingRoute: "local" as const, correctionSourceJobId: "job-base", correctionMode: "semantic_repair" as const, correctionRecipeSha256: "b".repeat(64), correctionFeedback: "Fix the part." };
    localStorage.setItem("device-studio:geometry-creation:p1", JSON.stringify({ request, sessionId: "session-1", phase: "draft" }));
    vi.mocked(getGeometryCreationSession).mockResolvedValue({ session_id: "session-1", state: "active", version: 2, policy: { project_id: "p1", candidate_id: "creation-base", brief: "Original brief", envelope_mm: request.envelopeMM, reference_media_ids: request.referenceMediaIDs, funding_route: "local", tokens_per_call: 4096 }, max_provider_calls: 2, provider_calls_reserved: 1, max_children: 2, children_admitted: 1, children: [{ job_id: "job-base", ordinal: 1, state: "failed", version: 4, candidate_id: "creation-base" }] } as never);
    vi.mocked(getGeometryCorrection).mockResolvedValue({ job_id: "job-base", project_id: "p1", disposition: "available", mode: "semantic_repair", envelope_mm: request.envelopeMM, reference_media_ids: request.referenceMediaIDs, issue: { code: "part_outside_envelope", part_id: "base", axis: "x", message: "outside" }, prior_recipe_json: "{}", recipe_sha256: request.correctionRecipeSha256, recipe_bytes: 2 });
    vi.mocked(refineGeometryCreationSession).mockResolvedValueOnce({ session_id: "session-1", state: "active", version: 2, policy: { project_id: "p1", candidate_id: "creation-base", brief: "Original brief", envelope_mm: request.envelopeMM, reference_media_ids: request.referenceMediaIDs, funding_route: "local", tokens_per_call: 4096 }, max_provider_calls: 2, provider_calls_reserved: 1, max_children: 2, children_admitted: 1, children: [{ job_id: "job-base", ordinal: 1, state: "failed", version: 4, candidate_id: "creation-base" }], needs_confirmation: true, changed_scope: ["brief"] } as never);
    vi.mocked(refineGeometryCreationSession).mockResolvedValueOnce({ session_id: "session-2", state: "active", version: 2, policy: { project_id: "p1", candidate_id: request.candidateId, brief: request.brief, envelope_mm: request.envelopeMM, reference_media_ids: request.referenceMediaIDs, funding_route: "local", tokens_per_call: 4096, parent_session_id: "session-1" }, max_provider_calls: 2, provider_calls_reserved: 1, max_children: 2, children_admitted: 1, children: [{ job_id: "job-new", ordinal: 1, state: "queued", version: 1, candidate_id: request.candidateId }] } as never);
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    expect(await screen.findByLabelText("I confirm this local AI request.")).not.toBeChecked();
    const submit = await screen.findByRole("button", { name: "Submit corrected draft" });
    expect(submit).toBeDisabled();
    await user.click(screen.getByLabelText("I confirm this local AI request."));
    await user.click(submit);
    expect(await screen.findByTestId("geometry-scope-expansion")).toHaveTextContent("No new job has been dispatched");
    expect(vi.mocked(refineGeometryCreationSession).mock.calls[0]?.[1].confirmScopeExpansion).toBe(false);
    await user.click(screen.getByRole("button", { name: "Confirm changed scope and start a linked session" }));
    await waitFor(() => expect(refineGeometryCreationSession).toHaveBeenCalledTimes(2));
    expect(vi.mocked(refineGeometryCreationSession).mock.calls[1]?.[1].confirmScopeExpansion).toBe(true);
    expect(localStorage.getItem("device-studio:geometry-creation:p1")).toContain("session-2");
  });

  it("starts a new identity only after an explicit terminal-draft reset", async () => {
    const user = userEvent.setup();
    const request = { candidateId: "creation-old", idempotencyKey: "request-old", brief: "Old device", envelopeMM: [78, 148, 12] as [number, number, number], referenceMediaIDs: ["image-1"], fundingRoute: "local" };
    localStorage.setItem("device-studio:geometry-creation:p1", JSON.stringify({ request, jobId: "job-1" }));
    vi.mocked(getGeometryCreation).mockResolvedValueOnce({ job_id: "job-1", project_id: "p1", candidate_id: request.candidateId, state: "succeeded", version: 4, review_status: "unreviewed" });
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    await user.click(await screen.findByRole("button", { name: "Start a new geometry draft" }));
    expect(localStorage.getItem("device-studio:geometry-creation:p1")).toBeNull();
    await user.type(screen.getByLabelText("Describe the device"), "A second device");
    await user.type(screen.getByLabelText("Width (mm)"), "78");
    await user.type(screen.getByLabelText("Height (mm)"), "148");
    await user.type(screen.getByLabelText("Depth (mm)"), "12");
    await user.click(screen.getByLabelText("device.png · image/png · 1024 bytes"));
    await user.click(screen.getByLabelText("I confirm this local AI request."));
    await user.click(screen.getByRole("button", { name: "Create geometry draft" }));
    await waitFor(() => expect(createGeometryCreationSession).toHaveBeenCalledTimes(1));
    expect(vi.mocked(createGeometryCreationSession).mock.calls[0]?.[1].idempotencyKey).not.toBe(request.idempotencyKey);
  });

  it("stages a verified failed response as a new draft and submits only after fresh confirmation", async () => {
    const user = userEvent.setup();
    const previous = { candidateId: "creation-old", idempotencyKey: "request-old", brief: "Compact field device", envelopeMM: [78, 148, 12] as [number, number, number], referenceMediaIDs: ["image-1"], fundingRoute: "local" as const };
    const recipe = '{"version":"device_geometry_recipe.v1"}';
    localStorage.setItem("device-studio:geometry-creation:p1", JSON.stringify({ request: previous, jobId: "job-failed" }));
    vi.mocked(getGeometryCreation).mockResolvedValue({ job_id: "job-failed", project_id: "p1", candidate_id: previous.candidateId, state: "failed", version: 4, review_status: "unreviewed" });
    vi.mocked(getGeometryCorrection).mockResolvedValue({ job_id: "job-failed", project_id: "p1", disposition: "available", envelope_mm: previous.envelopeMM, reference_media_ids: previous.referenceMediaIDs, issue: { code: "part_outside_envelope", part_id: "housing", axis: "y", message: "part exceeds the locked envelope" }, prior_recipe_json: recipe, recipe_sha256: "a".repeat(64), recipe_bytes: new TextEncoder().encode(recipe).length });
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    await user.click(await screen.findByRole("button", { name: "Use verified failed response in a new draft" }));
    expect(await screen.findByTestId("geometry-correction-draft")).toHaveTextContent("housing");
    expect(startGeometryCreation).not.toHaveBeenCalled();
    expect(screen.getByLabelText("I confirm this local AI request.")).not.toBeChecked();
    await user.click(screen.getByLabelText("I confirm this local AI request."));
    await user.click(screen.getByRole("button", { name: "Submit corrected draft" }));
    await waitFor(() => expect(startGeometryCreation).toHaveBeenCalledTimes(1));
    const submitted = vi.mocked(startGeometryCreation).mock.calls[0]?.[1];
    expect(submitted).toMatchObject({ brief: previous.brief, envelopeMM: previous.envelopeMM, referenceMediaIDs: previous.referenceMediaIDs, fundingRoute: "local", correctionSourceJobId: "job-failed", correctionRecipeSha256: "a".repeat(64) });
    expect(submitted?.candidateId).not.toBe(previous.candidateId);
    expect(submitted?.idempotencyKey).not.toBe(previous.idempotencyKey);
  });

  it("shows and carries the strict numeric bevel bound from a failed response into semantic repair", async () => {
    const user = userEvent.setup();
    const previous = { candidateId: "creation-thin", idempotencyKey: "request-thin", brief: "Generic unbranded screen", envelopeMM: [900, 645.45, 194.82] as [number, number, number], referenceMediaIDs: ["image-1"], fundingRoute: "local" as const };
    const recipe = '{"version":"device_geometry_recipe.v1"}';
    localStorage.setItem("device-studio:geometry-creation:p1", JSON.stringify({ request: previous, jobId: "job-thin-failed" }));
    vi.mocked(getGeometryCreation).mockResolvedValue({ job_id: "job-thin-failed", project_id: "p1", candidate_id: previous.candidateId, state: "failed", version: 5, review_status: "unreviewed" });
    vi.mocked(getGeometryCorrection).mockResolvedValue({
      job_id: "job-thin-failed", project_id: "p1", disposition: "available", mode: "semantic_repair", envelope_mm: previous.envelopeMM, reference_media_ids: previous.referenceMediaIDs,
      issue: { code: "bevel_exceeds_part_dimensions", part_id: "lower_bezel", bevel_mm: 1, minimum_dimension_mm: 2, max_bevel_exclusive_mm: 1, message: "bevel must be less than half the smallest part dimension" },
      prior_recipe_json: recipe, recipe_sha256: "c".repeat(64), recipe_bytes: new TextEncoder().encode(recipe).length,
    });
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    await user.click(await screen.findByRole("button", { name: "Use verified failed response in a new draft" }));
    const correctionDraft = await screen.findByTestId("geometry-correction-draft");
    expect(within(correctionDraft).getByRole("alert")).toHaveTextContent("lower_bezel has bevel 1 mm");
    expect(within(correctionDraft).getByRole("alert")).toHaveTextContent("strictly less than 1 mm");
    const guidance = screen.getByLabelText("Correction guidance (editable)") as HTMLTextAreaElement;
    expect(guidance.value).toContain("strictly less than 1 mm");
    expect(guidance.value).toContain("equality is invalid");
    expect(startGeometryCreation).not.toHaveBeenCalled();
    await user.click(screen.getByLabelText("I confirm this local AI request."));
    await user.click(screen.getByRole("button", { name: "Submit corrected draft" }));
    await waitFor(() => expect(startGeometryCreation).toHaveBeenCalledTimes(1));
    expect(vi.mocked(startGeometryCreation).mock.calls[0]?.[1]).toMatchObject({ correctionSourceJobId: "job-thin-failed", correctionMode: "semantic_repair", correctionRecipeSha256: "c".repeat(64) });
  });

  it("stages a successful recipe for bounded visual refinement with source inputs locked", async () => {
    const user = userEvent.setup();
    const previous = { candidateId: "creation-ok", idempotencyKey: "request-ok", brief: "Compact field device", envelopeMM: [78, 148, 12] as [number, number, number], referenceMediaIDs: ["image-1"], fundingRoute: "local" as const };
    const recipe = '{"version":"device_geometry_recipe.v1","operation":"construct"}';
    localStorage.setItem("device-studio:geometry-creation:p1", JSON.stringify({ request: previous, jobId: "job-succeeded", phase: "pending" }));
    vi.mocked(getGeometryCreation).mockResolvedValue({ job_id: "job-succeeded", project_id: "p1", candidate_id: previous.candidateId, state: "succeeded", version: 4, review_status: "unreviewed" });
    vi.mocked(getGeometryCorrection).mockResolvedValue({ job_id: "job-succeeded", project_id: "p1", disposition: "available", mode: "visual_refinement", envelope_mm: previous.envelopeMM, reference_media_ids: previous.referenceMediaIDs, prior_recipe_json: recipe, recipe_sha256: "b".repeat(64), recipe_bytes: new TextEncoder().encode(recipe).length });
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    await user.click(await screen.findByRole("button", { name: "Refine proportions from held-out views" }));
    expect(await screen.findByTestId("geometry-correction-draft")).toHaveTextContent("Refine the accepted recipe from held-out views");
    expect(startGeometryCreation).not.toHaveBeenCalled();
    expect(screen.getByLabelText("Describe the device")).toBeDisabled();
    expect(screen.getByLabelText("Width (mm)")).toBeDisabled();
    expect(screen.getByLabelText("device.png · image/png · 1024 bytes")).toBeDisabled();
    await user.type(screen.getByLabelText("Correction guidance (editable)"), "Use a narrower bezel and softer corner transitions.");
    await user.click(screen.getByLabelText("I confirm this local AI request."));
    await user.click(screen.getByRole("button", { name: "Submit corrected draft" }));
    await waitFor(() => expect(startGeometryCreation).toHaveBeenCalledTimes(1));
    const submitted = vi.mocked(startGeometryCreation).mock.calls[0]?.[1];
    expect(submitted).toMatchObject({ brief: previous.brief, envelopeMM: previous.envelopeMM, referenceMediaIDs: previous.referenceMediaIDs, fundingRoute: "local", correctionSourceJobId: "job-succeeded", correctionMode: "visual_refinement", correctionRecipeSha256: "b".repeat(64), correctionFeedback: "Use a narrower bezel and softer corner transitions." });
    expect(submitted?.candidateId).not.toBe(previous.candidateId);
    expect(submitted?.idempotencyKey).not.toBe(previous.idempotencyKey);
  });
});
