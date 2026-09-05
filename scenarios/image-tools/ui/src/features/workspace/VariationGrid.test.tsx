/**
 * VariationGrid tests — the N-slot result grid: pending skeleton slots while a
 * run is in flight, resolved images with send/download actions once results
 * land, and nothing at all when idle with no results.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { VariationGrid } from "./VariationGrid";
import type { CreateVariation } from "./useCreate";

const makeVariation = (index: number): CreateVariation => ({
  index,
  result: {
    kind: "image",
    url: `blob:v${index}`,
    width: 512,
    height: 512,
    format: "png",
    jobId: "gen-1",
  },
  outputFile: new File(["x"], `v${index}.png`, { type: "image/png" }),
});

beforeEach(async () => {
  await setLocale("en");
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("VariationGrid", () => {
  it("renders pending slots sized to the requested count while busy", () => {
    renderWithProviders(
      <VariationGrid
        results={[]}
        requestedCount={4}
        busy
        onSendToCanvas={vi.fn()}
        onSendToEnhance={vi.fn()}
      />,
    );
    expect(screen.getByTestId(selectors.workspace.createVariation({ index: 1 }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workspace.createVariation({ index: 4 }))).toBeInTheDocument();
    expect(screen.queryByRole("img")).not.toBeInTheDocument();
  });

  it("renders nothing when idle with no results", () => {
    const { container } = renderWithProviders(
      <VariationGrid
        results={[]}
        requestedCount={4}
        busy={false}
        onSendToCanvas={vi.fn()}
        onSendToEnhance={vi.fn()}
      />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("renders each resolved variation and sends the chosen one to the canvas", async () => {
    const user = userEvent.setup();
    const onSendToCanvas = vi.fn();
    const results = [makeVariation(0), makeVariation(1)];
    renderWithProviders(
      <VariationGrid
        results={results}
        requestedCount={2}
        busy={false}
        onSendToCanvas={onSendToCanvas}
        onSendToEnhance={vi.fn()}
      />,
    );

    expect(screen.getAllByRole("img")).toHaveLength(2);
    await user.click(screen.getByTestId(selectors.workspace.createSend({ index: 2 })));
    expect(onSendToCanvas).toHaveBeenCalledWith(results[1]);
  });

  it("routes the chosen variation to Enhance via its send-to-enhance action", async () => {
    const user = userEvent.setup();
    const onSendToEnhance = vi.fn();
    const results = [makeVariation(0)];
    renderWithProviders(
      <VariationGrid
        results={results}
        requestedCount={1}
        busy={false}
        onSendToCanvas={vi.fn()}
        onSendToEnhance={onSendToEnhance}
      />,
    );

    await user.click(screen.getByRole("button", { name: /enhance/i }));
    expect(onSendToEnhance).toHaveBeenCalledWith(results[0]);
  });

  it("offers a download link with the variation's format in the filename", () => {
    renderWithProviders(
      <VariationGrid
        results={[makeVariation(0)]}
        requestedCount={1}
        busy={false}
        onSendToCanvas={vi.fn()}
        onSendToEnhance={vi.fn()}
      />,
    );

    const link = screen.getByRole("link", { name: /download/i });
    expect(link).toHaveAttribute("download", "variation-1.png");
    expect(link).toHaveAttribute("href", "blob:v0");
  });

  it("falls back to a png download filename when the result format is empty", () => {
    const variation = makeVariation(0);
    variation.result.format = "";
    renderWithProviders(
      <VariationGrid
        results={[variation]}
        requestedCount={1}
        busy={false}
        onSendToCanvas={vi.fn()}
        onSendToEnhance={vi.fn()}
      />,
    );

    expect(screen.getByRole("link", { name: /download/i })).toHaveAttribute(
      "download",
      "variation-1.png",
    );
  });

  it("mixes resolved and pending slots when fewer results than requested have landed", () => {
    renderWithProviders(
      <VariationGrid
        results={[makeVariation(0)]}
        requestedCount={3}
        busy
        onSendToCanvas={vi.fn()}
        onSendToEnhance={vi.fn()}
      />,
    );

    // Only the requested-count drives the skeleton when busy; here a single
    // result has landed so the resolved slot wins and the rest stay pending.
    expect(screen.getAllByRole("img")).toHaveLength(1);
    expect(screen.getByTestId(selectors.workspace.createVariation({ index: 1 }))).toBeInTheDocument();
  });
});
