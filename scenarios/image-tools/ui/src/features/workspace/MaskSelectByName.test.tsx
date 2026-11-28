import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { JobSchema } from "@vrooli/proto-types/image-tools/v1/jobs/jobs_pb";

import { JobState } from "../../api/jobs";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";
import { MaskBrush } from "./MaskBrush";
import type { NamedSelectionClient } from "./useNamedSelection";

const photo = new File(["photo"], "photo.jpg", { type: "image/jpeg" });
const accepted = { jobId: "select-job", modelId: "grounded-sam-2", tier: "local-gpu", warnings: [], estimatedSeconds: 3 };

const makeClient = (over: Partial<NamedSelectionClient> = {}): NamedSelectionClient => ({
  submit: vi.fn().mockResolvedValue(accepted),
  wait: vi.fn().mockResolvedValue(create(JobSchema, { id: accepted.jobId, state: JobState.SUCCEEDED, resultRef: "out/mask.png" })),
  cancel: vi.fn().mockResolvedValue(undefined),
  fetchMask: vi.fn().mockResolvedValue(new Blob(["png"], { type: "image/png" })),
  ...over,
});

afterEach(() => cleanup());

describe("MaskBrush select by name", () => {
  it("a quick choice selects that part and hands the mask to the edit", async () => {
    const client = makeClient();
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl={null} image={photo} onMask={onMask} selectionClient={client} />);
    await userEvent.click(screen.getByTestId(selectors.workspace.maskQuickSelect({ phrase: "hair" })));
    await waitFor(() => expect(onMask).toHaveBeenCalledTimes(1));
    expect(client.submit).toHaveBeenCalledWith("segment", { opParams: { segmentation: { text: "hair" } } }, expect.objectContaining({ image: photo }));
    const mask = onMask.mock.calls[0]?.[0] as File;
    expect(mask).toBeInstanceOf(File);
    expect(screen.getByTestId(selectors.workspace.mask.status).textContent).toBe(strings.workspace.mask.selected);
  });

  it("a typed phrase selects on submit, and Clear forgets the selection", async () => {
    const client = makeClient();
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl={null} image={photo} onMask={onMask} selectionClient={client} />);
    await userEvent.type(screen.getByTestId(selectors.workspace.mask.selectInput), "the book{enter}");
    await waitFor(() => expect(onMask).toHaveBeenCalledTimes(1));
    expect(client.submit).toHaveBeenCalledWith("segment", { opParams: { segmentation: { text: "the book" } } }, expect.anything());
    await userEvent.click(screen.getByTestId(selectors.workspace.mask.clear));
    expect(onMask).toHaveBeenLastCalledWith(null);
    expect(screen.getByTestId(selectors.workspace.mask.status).textContent).toBe(strings.workspace.mask.empty);
  });

  it("says why a phrase found nothing and leaves the mask alone", async () => {
    const reason = "nothing in the image matched 'unicorn'";
    const client = makeClient({ wait: vi.fn().mockResolvedValue(create(JobSchema, { id: accepted.jobId, state: JobState.FAILED, error: reason })) });
    const onMask = vi.fn();
    renderWithProviders(<MaskBrush imageUrl={null} image={photo} onMask={onMask} selectionClient={client} />);
    await userEvent.type(screen.getByTestId(selectors.workspace.mask.selectInput), "unicorn{enter}");
    expect((await screen.findByTestId(selectors.workspace.mask.selectError)).textContent).toBe(reason);
    expect(onMask).not.toHaveBeenCalled();
  });

  it("offers nothing to select in until there is a picture", () => {
    renderWithProviders(<MaskBrush imageUrl={null} image={null} onMask={vi.fn()} selectionClient={makeClient()} />);
    expect(screen.getByTestId(selectors.workspace.mask.selectRun).hasAttribute("disabled")).toBe(true);
    expect(screen.getByTestId(selectors.workspace.maskQuickSelect({ phrase: "hair" })).hasAttribute("disabled")).toBe(true);
  });
});
