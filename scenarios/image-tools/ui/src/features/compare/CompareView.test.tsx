import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { DiffResultSchema, type DiffResult } from "@vrooli/proto-types/image-tools/v1/diff/diff_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { makeApiError } from "../../api/client";
import { DiffMode } from "../../api/diff";
import { CompareView } from "./CompareView";
import type { CompareClient } from "./useCompare";

const diffResult = (over: Partial<DiffResult> = {}): DiffResult =>
  create(DiffResultSchema, {
    jobId: "job-diff",
    verdict: "different",
    dimensionsMatch: true,
    baseWidth: 100,
    baseHeight: 80,
    compareWidth: 100,
    compareHeight: 80,
    changedPixels: 1234n,
    totalPixels: 8000n,
    changedFraction: 0.154,
    mae: 12.5,
    rmse: 20.1,
    psnr: 28.3,
    phashDistance: 7,
    phashSimilarity: 0.89,
    ssim: 0.74,
    heatmapRef: "out/heat.png",
    warnings: [],
    ...over,
  });

const fakeClient = (over: Partial<CompareClient> = {}): CompareClient => ({
  compare: vi.fn().mockResolvedValue(diffResult()),
  blobUrl: (key: string) => `/api/v1/blobs/${key}`,
  ...over,
});

const loadBoth = async () => {
  await userEvent.upload(
    screen.getByTestId<HTMLInputElement>(selectors.compare.baseFileInput),
    new File(["base"], "base.png", { type: "image/png" }),
  );
  await userEvent.upload(
    screen.getByTestId<HTMLInputElement>(selectors.compare.compareFileInput),
    new File(["compare"], "compare.png", { type: "image/png" }),
  );
};

beforeEach(() => {
  vi.stubGlobal("URL", {
    ...URL,
    createObjectURL: vi.fn(() => "blob:fake"),
    revokeObjectURL: vi.fn(),
  });
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

describe("CompareView", () => {
  it("disables Compare until both images are chosen (file-picker path)", async () => {
    renderWithProviders(<CompareView client={fakeClient()} />);
    expect(screen.getByTestId(selectors.compare.runButton)).toBeDisabled();
    await loadBoth();
    expect(screen.getByTestId(selectors.compare.runButton)).toBeEnabled();
    expect(screen.getByTestId(selectors.compare.baseImage)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.compare.compareImage)).toBeInTheDocument();
  });

  it("runs a comparison and renders verdict, heat-map, metrics", async () => {
    const client = fakeClient();
    renderWithProviders(<CompareView client={client} />);
    await loadBoth();
    await userEvent.click(screen.getByTestId(selectors.compare.runButton));

    await waitFor(() => expect(screen.getByTestId(selectors.compare.result)).toBeInTheDocument());
    expect(client.compare).toHaveBeenCalled();
    expect(screen.getByTestId(selectors.compare.verdict)).toHaveTextContent(strings.compare.verdictDifferent);
    expect(screen.getByTestId(selectors.compare.heatmap)).toHaveAttribute("src", "/api/v1/blobs/out/heat.png");
    expect(screen.getByTestId(selectors.compare.metrics)).toBeInTheDocument();
  });

  it("threads the perceptual mode + tolerance into the compare call", async () => {
    const client = fakeClient();
    renderWithProviders(<CompareView client={client} />);
    await loadBoth();
    await userEvent.click(screen.getByTestId(selectors.compare.modeOption({ mode: "perceptual" })));
    await userEvent.click(screen.getByTestId(selectors.compare.runButton));

    await waitFor(() => expect(client.compare).toHaveBeenCalled());
    const call = (client.compare as ReturnType<typeof vi.fn>).mock.calls[0]?.[0];
    expect(call.mode).toBe(DiffMode.PERCEPTUAL);
  });

  it("renders no heat-map figure when heatmapRef is empty", async () => {
    const client = fakeClient({ compare: vi.fn().mockResolvedValue(diffResult({ heatmapRef: "" })) });
    renderWithProviders(<CompareView client={client} />);
    await loadBoth();
    await userEvent.click(screen.getByTestId(selectors.compare.runButton));
    await waitFor(() => expect(screen.getByTestId(selectors.compare.result)).toBeInTheDocument());
    expect(screen.queryByTestId(selectors.compare.heatmap)).not.toBeInTheDocument();
  });

  it("shows warnings when the result carries them", async () => {
    const client = fakeClient({
      compare: vi.fn().mockResolvedValue(diffResult({ dimensionsMatch: false, warnings: ["images differ in size"] })),
    });
    renderWithProviders(<CompareView client={client} />);
    await loadBoth();
    await userEvent.click(screen.getByTestId(selectors.compare.runButton));
    await waitFor(() => expect(screen.getByTestId(selectors.compare.warnings)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.compare.warnings)).toHaveTextContent("images differ in size");
  });

  it("renders the identical and similar verdicts", async () => {
    const client = fakeClient({ compare: vi.fn().mockResolvedValue(diffResult({ verdict: "identical", heatmapRef: "" })) });
    renderWithProviders(<CompareView client={client} />);
    await loadBoth();
    await userEvent.click(screen.getByTestId(selectors.compare.runButton));
    await waitFor(() =>
      expect(screen.getByTestId(selectors.compare.verdict)).toHaveTextContent(strings.compare.verdictIdentical),
    );

    cleanup();
    const similar = fakeClient({ compare: vi.fn().mockResolvedValue(diffResult({ verdict: "similar", heatmapRef: "" })) });
    renderWithProviders(<CompareView client={similar} />);
    await loadBoth();
    await userEvent.click(screen.getByTestId(selectors.compare.runButton));
    await waitFor(() =>
      expect(screen.getByTestId(selectors.compare.verdict)).toHaveTextContent(strings.compare.verdictSimilar),
    );
  });

  it("surfaces a failed comparison as an error", async () => {
    const client = fakeClient({ compare: vi.fn().mockRejectedValue(makeApiError("internal", "boom", 500)) });
    renderWithProviders(<CompareView client={client} />);
    await loadBoth();
    await userEvent.click(screen.getByTestId(selectors.compare.runButton));
    await waitFor(() => expect(screen.getByTestId(selectors.compare.error)).toBeInTheDocument());
    expect(screen.queryByTestId(selectors.compare.result)).not.toBeInTheDocument();
  });

  it("reset clears a prior verdict", async () => {
    renderWithProviders(<CompareView client={fakeClient()} />);
    await loadBoth();
    await userEvent.click(screen.getByTestId(selectors.compare.runButton));
    await waitFor(() => expect(screen.getByTestId(selectors.compare.result)).toBeInTheDocument());
    await userEvent.click(screen.getByTestId(selectors.compare.reset));
    expect(screen.queryByTestId(selectors.compare.result)).not.toBeInTheDocument();
  });
});
