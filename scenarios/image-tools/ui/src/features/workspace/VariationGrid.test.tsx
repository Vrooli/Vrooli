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
});
