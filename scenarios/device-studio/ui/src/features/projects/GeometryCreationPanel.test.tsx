import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { MediaReadiness, SourceMediaKind, SourceMediaSchema } from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { cancelGeometryCreation, findGeometryCreationOperation, getGeometryCreation, startGeometryCreation } from "../../api/creation";
import { projectsClient } from "../../api/projects";
import { GeometryCreationPanel } from "./GeometryCreationPanel";

vi.mock("../../api/projects", () => ({ projectsClient: { listSourceMedia: vi.fn() } }));
vi.mock("../../api/creation", () => ({ startGeometryCreation: vi.fn(), findGeometryCreationOperation: vi.fn(), getGeometryCreation: vi.fn(), cancelGeometryCreation: vi.fn() }));

const media = create(SourceMediaSchema, { id: "image-1", projectId: "p1", originalFilename: "device.png", mediaType: "image/png", sha256: "a".repeat(64), sizeBytes: 1024n, kind: SourceMediaKind.IMAGE, readiness: MediaReadiness.READY, width: 800, height: 1200 });

describe("GeometryCreationPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
    localStorage.clear();
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [media], nextPageToken: "" } as never);
    vi.mocked(findGeometryCreationOperation).mockResolvedValue(null);
    vi.mocked(startGeometryCreation).mockResolvedValue({ job_id: "job-1", project_id: "p1", candidate_id: "creation-1", state: "queued", version: 1, review_status: "unreviewed" });
    vi.mocked(getGeometryCreation).mockResolvedValue({ job_id: "job-1", project_id: "p1", candidate_id: "creation-1", state: "queued", version: 1, review_status: "unreviewed" });
    vi.mocked(cancelGeometryCreation).mockResolvedValue({ job_id: "job-1", project_id: "p1", candidate_id: "creation-1", state: "cancel_requested", version: 2, review_status: "unreviewed" });
  });
  afterEach(() => { cleanup(); vi.clearAllMocks(); });

  it("requires clear local-AI consent, bounded dimensions and selected project image before dispatch", async () => {
    const user = userEvent.setup();
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    const createButton = await screen.findByRole("button", { name: "Create local geometry draft" });
    expect(createButton).toBeDisabled();
    await user.type(screen.getByLabelText("Describe the device"), "A compact field phone with physical controls");
    await user.type(screen.getByLabelText("Width (mm)"), "78");
    await user.type(screen.getByLabelText("Height (mm)"), "148");
    await user.type(screen.getByLabelText("Depth (mm)"), "12");
    await user.click(screen.getByLabelText("device.png · image/png · 1024 bytes"));
    expect(createButton).toBeDisabled();
    await user.click(screen.getByLabelText(/I understand the selected images/));
    expect(createButton).toBeEnabled();
    await user.click(createButton);
    await waitFor(() => expect(startGeometryCreation).toHaveBeenCalledWith("p1", expect.objectContaining({ brief: "A compact field phone with physical controls", envelopeMM: [78, 148, 12], referenceMediaIDs: ["image-1"] })));
    expect(await screen.findByTestId("geometry-creation-status")).toHaveTextContent("unreviewed");
  });

  it("does not enable hosted fallback or call the job owner when local images are unavailable", async () => {
    vi.mocked(projectsClient.listSourceMedia).mockRejectedValueOnce(new Error("offline"));
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Project images could not be loaded.");
    expect(startGeometryCreation).not.toHaveBeenCalled();
    expect(screen.getByText(/No hosted fallback is allowed/)).toBeInTheDocument();
  });

  it("recovers an uncertain submit after remount and retries with the same durable identity", async () => {
    const user = userEvent.setup();
    vi.mocked(startGeometryCreation).mockRejectedValueOnce(new Error("connection dropped"));
    const first = renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    await user.type(await screen.findByLabelText("Describe the device"), "Compact field device");
    await user.type(screen.getByLabelText("Width (mm)"), "78");
    await user.type(screen.getByLabelText("Height (mm)"), "148");
    await user.type(screen.getByLabelText("Depth (mm)"), "12");
    await user.click(screen.getByLabelText("device.png · image/png · 1024 bytes"));
    await user.click(screen.getByLabelText(/I understand the selected images/));
    await user.click(screen.getByRole("button", { name: "Create local geometry draft" }));
    await screen.findByRole("alert");
    const firstRequest = vi.mocked(startGeometryCreation).mock.calls[0]?.[1];
    expect(firstRequest).toBeDefined();
    first.unmount();

    vi.mocked(startGeometryCreation).mockResolvedValueOnce({ job_id: "job-recovered", project_id: "p1", candidate_id: firstRequest!.candidateId, state: "queued", version: 1, review_status: "unreviewed" });
    vi.mocked(getGeometryCreation).mockResolvedValueOnce({ job_id: "job-recovered", project_id: "p1", candidate_id: firstRequest!.candidateId, state: "queued", version: 1, review_status: "unreviewed" });
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    const retry = await screen.findByRole("button", { name: "Retry the same request" });
    await user.click(retry);
    await waitFor(() => expect(startGeometryCreation).toHaveBeenCalledTimes(2));
    expect(vi.mocked(startGeometryCreation).mock.calls[1]).toEqual(["p1", firstRequest]);
    expect(findGeometryCreationOperation).toHaveBeenCalledWith("p1", firstRequest!.idempotencyKey);
    expect(await screen.findByTestId("geometry-creation-status")).toHaveTextContent("job-recovered");
  });

  it("restores owner-confirmed status after reload without starting another request", async () => {
    const request = { candidateId: "creation-stable", idempotencyKey: "request-stable", brief: "Compact field device", envelopeMM: [78, 148, 12] as [number, number, number], referenceMediaIDs: ["image-1"] };
    localStorage.setItem("device-studio:geometry-creation:p1", JSON.stringify({ request }));
    vi.mocked(findGeometryCreationOperation).mockResolvedValueOnce({ job_id: "job-owner", project_id: "p1", candidate_id: request.candidateId, state: "running", version: 2, review_status: "unreviewed" });
    vi.mocked(getGeometryCreation).mockResolvedValueOnce({ job_id: "job-owner", project_id: "p1", candidate_id: request.candidateId, state: "running", version: 2, review_status: "unreviewed" });
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    expect(await screen.findByTestId("geometry-creation-status")).toHaveTextContent("job-owner");
    expect(findGeometryCreationOperation).toHaveBeenCalledWith("p1", request.idempotencyKey);
    expect(startGeometryCreation).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: "Create local geometry draft" })).not.toBeInTheDocument();
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
    await user.click(screen.getByLabelText(/I understand the selected images/));
    await user.click(screen.getByRole("button", { name: "Create local geometry draft" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Browser storage is unavailable");
    expect(startGeometryCreation).not.toHaveBeenCalled();
    first.unmount();

    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    expect(await screen.findByRole("button", { name: "Create local geometry draft" })).toBeDisabled();
    expect(startGeometryCreation).not.toHaveBeenCalled();
    setItem.mockRestore();
  });

  it("routes cancellation through the job owner for a recovered active operation", async () => {
    const user = userEvent.setup();
    const request = { candidateId: "creation-stable", idempotencyKey: "request-stable", brief: "Compact field device", envelopeMM: [78, 148, 12] as [number, number, number], referenceMediaIDs: ["image-1"] };
    localStorage.setItem("device-studio:geometry-creation:p1", JSON.stringify({ request, jobId: "job-1" }));
    vi.mocked(getGeometryCreation).mockResolvedValueOnce({ job_id: "job-1", project_id: "p1", candidate_id: request.candidateId, state: "running", version: 3, review_status: "unreviewed" });
    renderWithProviders(<GeometryCreationPanel projectId="p1" />);
    await user.click(await screen.findByRole("button", { name: "Request cancellation" }));
    await waitFor(() => expect(cancelGeometryCreation).toHaveBeenCalledWith(expect.objectContaining({ job_id: "job-1", version: 3 })));
    expect(await screen.findByTestId("geometry-creation-status")).toHaveTextContent("cancel_requested");
  });

  it("starts a new identity only after an explicit terminal-draft reset", async () => {
    const user = userEvent.setup();
    const request = { candidateId: "creation-old", idempotencyKey: "request-old", brief: "Old device", envelopeMM: [78, 148, 12] as [number, number, number], referenceMediaIDs: ["image-1"] };
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
    await user.click(screen.getByLabelText(/I understand the selected images/));
    await user.click(screen.getByRole("button", { name: "Create local geometry draft" }));
    await waitFor(() => expect(startGeometryCreation).toHaveBeenCalledTimes(1));
    expect(vi.mocked(startGeometryCreation).mock.calls[0]?.[1].idempotencyKey).not.toBe(request.idempotencyKey);
  });
});
