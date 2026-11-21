import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import {
  SegmentResultSchema,
  SuggestedEditSchema,
  type SegmentResult,
} from "@vrooli/proto-types/image-tools/v1/selection/selection_pb";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { makeApiError } from "../../api/client";
import { SmartSelectView } from "./SmartSelectView";
import type { SmartSelectClient } from "./useSmartSelect";

const objectResult = (): SegmentResult =>
  create(SegmentResultSchema, {
    jobId: "job-segment",
    maskRef: "mask/abc.png",
    box: { x: 0.3, y: 0.3, width: 0.4, height: 0.4 },
    regionClass: "object",
    confidence: 0.62,
    areaFraction: 0.16,
    tier: "builtin-cpu",
    suggestedEdits: [
      create(SuggestedEditSchema, { id: "remove", label: "Remove this object", description: "Remove it", operation: "object_removal", requiresMask: true }),
      create(SuggestedEditSchema, { id: "replace", label: "Replace with…", description: "Regenerate", operation: "inpaint", requiresPrompt: true, requiresMask: true }),
    ],
  });

const fakeClient = (over: Partial<SmartSelectClient> = {}): SmartSelectClient => ({
  segment: vi.fn().mockResolvedValue(objectResult()),
  fetchBlob: vi.fn().mockResolvedValue(new Blob(["mask"], { type: "image/png" })),
  blobUrl: (key: string) => `/api/v1/blobs/${key}`,
  submitAI: vi.fn().mockResolvedValue({ jobId: "job-ai", estimatedSeconds: 5, modelId: "m", tier: "local-cpu", warnings: [] }),
  ...over,
});

const loadImage = async () => {
  const file = new File(["img"], "in.png", { type: "image/png" });
  const input = screen.getByTestId<HTMLInputElement>(selectors.select.fileInput);
  await userEvent.upload(input, file);
};

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("SmartSelectView", () => {
  it("loads an image then segments via the auto-select fallback", async () => {
    const client = fakeClient();
    renderWithProviders(<SmartSelectView client={client} />);
    await loadImage();

    await userEvent.click(screen.getByTestId(selectors.select.autoButton));
    await waitFor(() => expect(client.segment).toHaveBeenCalled());
    expect(screen.getByTestId(selectors.select.regionClass)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.select.maskOverlay)).toBeInTheDocument();
  });

  it("segments from the X/Y coordinate fallback (no-pointer path)", async () => {
    const client = fakeClient();
    renderWithProviders(<SmartSelectView client={client} />);
    await loadImage();

    await userEvent.clear(screen.getByTestId(selectors.select.pointXInput));
    await userEvent.type(screen.getByTestId(selectors.select.pointXInput), "25");
    await userEvent.click(screen.getByTestId(selectors.select.selectPointButton));

    await waitFor(() => expect(client.segment).toHaveBeenCalled());
    const call = (client.segment as ReturnType<typeof vi.fn>).mock.calls[0]?.[0];
    expect(call.mode).toBeDefined();
    expect(call.points?.[0].x).toBeCloseTo(0.25);
  });

  it("applies a no-prompt edit as a masked AI submit", async () => {
    const client = fakeClient();
    renderWithProviders(<SmartSelectView client={client} />);
    await loadImage();
    await userEvent.click(screen.getByTestId(selectors.select.autoButton));
    await waitFor(() => expect(screen.getByTestId(selectors.select.regionClass)).toBeInTheDocument());

    await userEvent.click(screen.getByTestId(selectors.select.editButton({ id: "remove" })));
    await waitFor(() => expect(client.submitAI).toHaveBeenCalledWith("object_removal", expect.anything(), expect.objectContaining({ mask: expect.any(File) })));
    expect(client.fetchBlob).toHaveBeenCalledWith("mask/abc.png");
    expect(screen.getByTestId(selectors.select.applyResult)).toBeInTheDocument();
  });

  it("fills a requires-prompt edit's instruction before submitting", async () => {
    const client = fakeClient();
    renderWithProviders(<SmartSelectView client={client} />);
    await loadImage();
    await userEvent.click(screen.getByTestId(selectors.select.autoButton));
    await waitFor(() => expect(screen.getByTestId(selectors.select.regionClass)).toBeInTheDocument());

    await userEvent.type(screen.getByTestId(selectors.select.promptInput), "a red ball");
    await userEvent.click(screen.getByTestId(selectors.select.editButton({ id: "replace" })));
    await waitFor(() => expect(client.submitAI).toHaveBeenCalled());
    const params = (client.submitAI as ReturnType<typeof vi.fn>).mock.calls[0]?.[1];
    expect(params.prompt).toContain("a red ball");
  });

  it("surfaces a 409 as an actionable gate, not an opaque failure", async () => {
    const gated = fakeClient({
      submitAI: vi.fn().mockRejectedValue(makeApiError("invalid_request", "model not installed", 409)),
    });
    renderWithProviders(<SmartSelectView client={gated} />);
    await loadImage();
    await userEvent.click(screen.getByTestId(selectors.select.autoButton));
    await waitFor(() => expect(screen.getByTestId(selectors.select.regionClass)).toBeInTheDocument());

    await userEvent.click(screen.getByTestId(selectors.select.editButton({ id: "remove" })));
    await waitFor(() => expect(screen.getByTestId(selectors.select.applyResult)).toBeInTheDocument());
    // cimode renders the i18n *key*, so assert the gate string resolves to the
    // gate copy key (not an opaque error message).
    expect(screen.getByText(strings.select.gate)).toBeInTheDocument();
  });

  it("has no a11y violations once a selection is made", async () => {
    const { container } = renderWithProviders(<SmartSelectView client={fakeClient()} />);
    await loadImage();
    await userEvent.click(screen.getByTestId(selectors.select.autoButton));
    await waitFor(() => expect(screen.getByTestId(selectors.select.regionClass)).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });
});
