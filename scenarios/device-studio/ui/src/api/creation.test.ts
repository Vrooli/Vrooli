import { afterEach, describe, expect, it, vi } from "vitest";
import { cancelGeometryCreation, findGeometryCreationOperation, getGeometryCorrection, getGeometryCreation, startGeometryCreation } from "./creation";

const mocks = vi.hoisted(() => ({ fileURL: vi.fn((path: string) => `http://device.test/api/v1${path}`), getFile: vi.fn() }));
vi.mock("./client", () => mocks);

describe("geometry creation API boundary", () => {
  afterEach(() => { vi.unstubAllGlobals(); vi.clearAllMocks(); });

  it("requires explicit bounded request inputs and sends the selected route and disclosure", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ job_id: "job-1", project_id: "project-1", candidate_id: "candidate-1", state: "queued", version: 1, review_status: "unreviewed" }), { status: 202 }));
    vi.stubGlobal("fetch", fetchMock);
    const result = await startGeometryCreation("project/one", { candidateId: "candidate-1", idempotencyKey: "key-1", brief: "compact field device", envelopeMM: [80, 150, 12], referenceMediaIDs: ["media-1"], fundingRoute: "local" });
    expect(result.state).toBe("queued");
    expect(fetchMock).toHaveBeenCalledWith("http://device.test/api/v1/projects/project%2Fone/geometry-creations", expect.objectContaining({ method: "POST", cache: "no-store" }));
    const body = JSON.parse((fetchMock.mock.calls[0]?.[1] as RequestInit).body as string);
    expect(body.confirm_local_ai).toBe(true);
    expect(body.funding_route).toBe("local");
    expect(body.confirm_remote_ai).toBe(false);
    expect(body.reference_media_ids).toEqual(["media-1"]);
    await expect(startGeometryCreation("project", { candidateId: "c", idempotencyKey: "k", brief: "x", envelopeMM: [80, 150, 12], referenceMediaIDs: [], fundingRoute: "local" })).rejects.toThrow("one to four project images");
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
    await expect(startGeometryCreation("p", { candidateId: "c", idempotencyKey: "k", brief: "device", envelopeMM: [80, 150, 12], referenceMediaIDs: ["m"], fundingRoute: "byok" })).rejects.toThrow("incomplete job status");
  });

  it("accepts only the exact job-bound retained correction response and rejects altered bytes", async () => {
    const recipe = '{"version":"device_geometry_recipe.v1"}';
    const digest = [...new Uint8Array(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(recipe)))].map((value) => value.toString(16).padStart(2, "0")).join("");
    const details = { job_id: "job-1", project_id: "p", disposition: "available", issue: { code: "part_outside_envelope", part_id: "housing", axis: "y", message: "part exceeds the locked envelope" }, envelope_mm: [80, 150, 12], prior_recipe_json: recipe, recipe_sha256: digest, recipe_bytes: new TextEncoder().encode(recipe).length };
    mocks.getFile.mockResolvedValueOnce(new Response(JSON.stringify(details), { status: 200 }));
    await expect(getGeometryCorrection("job-1")).resolves.toMatchObject({ recipe_sha256: digest, prior_recipe_json: recipe });
    expect(mocks.getFile).toHaveBeenCalledWith("/geometry-creations/job-1/correction");
    mocks.getFile.mockResolvedValueOnce(new Response(JSON.stringify({ ...details, prior_recipe_json: recipe.replace("device", "Device") }), { status: 200 }));
    await expect(getGeometryCorrection("job-1")).rejects.toThrow("digest differs");
  });
});
