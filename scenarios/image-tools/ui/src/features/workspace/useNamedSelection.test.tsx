import { act, renderHook } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { JobSchema } from "@vrooli/proto-types/image-tools/v1/jobs/jobs_pb";
import { afterEach, expect, it, vi } from "vitest";

import { JobState } from "../../api/jobs";
import { useNamedSelection, type NamedSelectionClient } from "./useNamedSelection";

const photo = new File(["photo"], "photo.jpg", { type: "image/jpeg" });
const accepted = { jobId: "select-job", modelId: "grounded-sam-2", tier: "local-gpu", warnings: [], estimatedSeconds: 3 };

const makeClient = (over: Partial<NamedSelectionClient> = {}): NamedSelectionClient => ({
  submit: vi.fn().mockResolvedValue(accepted),
  wait: vi.fn().mockResolvedValue(create(JobSchema, { id: accepted.jobId, state: JobState.SUCCEEDED, resultRef: "out/mask.png" })),
  cancel: vi.fn().mockResolvedValue(undefined),
  fetchMask: vi.fn().mockResolvedValue(new Blob(["png"], { type: "image/png" })),
  ...over,
});

afterEach(() => vi.clearAllMocks());

it("selects by phrase with the default segmentation model and returns the mask as a PNG file", async () => {
  const client = makeClient();
  const { result } = renderHook(() => useNamedSelection(client));
  let mask: File | null = null;
  await act(async () => { mask = await result.current.select(photo, "  hair "); });
  expect(client.submit).toHaveBeenCalledWith(
    "segment",
    { opParams: { segmentation: { text: "hair" } } },
    expect.objectContaining({ image: photo, operationId: expect.any(String) }),
  );
  expect(client.fetchMask).toHaveBeenCalledWith("out/mask.png");
  expect(mask).toBeInstanceOf(File);
  expect((mask as unknown as File).type).toBe("image/png");
  expect(result.current.busy).toBe(false);
  expect(result.current.error).toBe("");
});

it("does nothing for an empty phrase", async () => {
  const client = makeClient();
  const { result } = renderHook(() => useNamedSelection(client));
  let mask: File | null = photo;
  await act(async () => { mask = await result.current.select(photo, "   "); });
  expect(mask).toBeNull();
  expect(client.submit).not.toHaveBeenCalled();
});

it("shows the reason a phrase could not be selected, not a generic failure", async () => {
  const reason = "nothing in the image matched 'unicorn'; try a different or more specific phrase";
  const client = makeClient({
    wait: vi.fn().mockResolvedValue(create(JobSchema, { id: accepted.jobId, state: JobState.FAILED, error: reason })),
  });
  const { result } = renderHook(() => useNamedSelection(client));
  let mask: File | null = photo;
  await act(async () => { mask = await result.current.select(photo, "unicorn"); });
  expect(mask).toBeNull();
  expect(result.current.error).toBe(reason);
  expect(client.fetchMask).not.toHaveBeenCalled();
});

it("cancels the selection job when the user cancels while it is admitted", async () => {
  let release!: (value: typeof accepted) => void;
  const client = makeClient({ submit: vi.fn().mockImplementation(() => new Promise((resolve) => { release = resolve; })) });
  const { result } = renderHook(() => useNamedSelection(client));
  let pending!: Promise<File | null>;
  act(() => { pending = result.current.select(photo, "hair"); });
  act(() => result.current.cancel());
  await act(async () => { release(accepted); await pending; });
  expect(client.cancel).toHaveBeenCalledWith(accepted.jobId);
  expect(client.wait).not.toHaveBeenCalled();
  expect(result.current.busy).toBe(false);
});
