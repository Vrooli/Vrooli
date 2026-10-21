import { afterEach, describe, expect, it, vi } from "vitest";
import { cancelGeometryCreation, findGeometryCreationOperation, getGeometryCreation, startGeometryCreation } from "./creation";

const mocks = vi.hoisted(() => ({ fileURL: vi.fn((path: string) => `http://device.test/api/v1${path}`), getFile: vi.fn() }));
vi.mock("./client", () => mocks);

describe("geometry creation API boundary", () => {
  afterEach(() => { vi.unstubAllGlobals(); vi.clearAllMocks(); });

  it("requires explicit bounded request inputs and sends only the local-AI disclosure", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ job_id: "job-1", project_id: "project-1", candidate_id: "candidate-1", state: "queued", version: 1, review_status: "unreviewed" }), { status: 202 }));
    vi.stubGlobal("fetch", fetchMock);
    const result = await startGeometryCreation("project/one", { candidateId: "candidate-1", idempotencyKey: "key-1", brief: "compact field device", envelopeMM: [80, 150, 12], referenceMediaIDs: ["media-1"] });
    expect(result.state).toBe("queued");
    expect(fetchMock).toHaveBeenCalledWith("http://device.test/api/v1/projects/project%2Fone/geometry-creations", expect.objectContaining({ method: "POST", cache: "no-store" }));
    const body = JSON.parse((fetchMock.mock.calls[0]?.[1] as RequestInit).body as string);
    expect(body.confirm_local_ai).toBe(true);
    expect(body.reference_media_ids).toEqual(["media-1"]);
    await expect(startGeometryCreation("project", { candidateId: "c", idempotencyKey: "k", brief: "x", envelopeMM: [80, 150, 12], referenceMediaIDs: [] })).rejects.toThrow("one to four project images");
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("reads durable status and cancels using its exact observed version", async () => {
    const job = { job_id: "job/one", project_id: "p", candidate_id: "c", state: "running", version: 7, review_status: "unreviewed" };
    mocks.getFile.mockResolvedValueOnce(new Response(JSON.stringify(job), { status: 200 }));
    await expect(getGeometryCreation("job/one")).resolves.toMatchObject({ state: "running", version: 7 });
    expect(mocks.getFile).toHaveBeenCalledWith("/geometry-creations/job%2Fone");
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ ...job, state: "cancel_requested", version: 8 }), { status: 202 }));
    vi.stubGlobal("fetch", fetchMock);
    await expect(cancelGeometryCreation(job)).resolves.toMatchObject({ state: "cancel_requested" });
    expect(JSON.parse((fetchMock.mock.calls[0]?.[1] as RequestInit).body as string)).toEqual({ version: 7 });
  });

  it("reads an operation by its stable project-scoped idempotency key without submitting", async () => {
    const job = { job_id: "job-1", project_id: "project-1", candidate_id: "candidate-1", state: "queued", version: 1, review_status: "unreviewed" };
    const fetchMock = vi.fn().mockResolvedValueOnce(new Response(JSON.stringify(job), { status: 200 })).mockResolvedValueOnce(new Response("{}", { status: 404 }));
    vi.stubGlobal("fetch", fetchMock);
    await expect(findGeometryCreationOperation("project/one", "request/one")).resolves.toMatchObject({ job_id: "job-1" });
    await expect(findGeometryCreationOperation("project/one", "absent")).resolves.toBeNull();
    expect(fetchMock).toHaveBeenNthCalledWith(1, "http://device.test/api/v1/projects/project%2Fone/geometry-creations/operations/request%2Fone", { cache: "no-store" });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("does not accept successful responses that promote generated geometry", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ job_id: "job", project_id: "p", candidate_id: "c", state: "succeeded", version: 2, review_status: "approved" }), { status: 202 })));
    await expect(startGeometryCreation("p", { candidateId: "c", idempotencyKey: "k", brief: "device", envelopeMM: [80, 150, 12], referenceMediaIDs: ["m"] })).rejects.toThrow("incomplete job status");
  });
});
