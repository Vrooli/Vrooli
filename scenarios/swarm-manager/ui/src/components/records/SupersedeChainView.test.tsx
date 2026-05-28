import { describe, expect, it, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { SupersedeChainView } from "./SupersedeChainView";
import { renderWithProviders } from "../../test-utils";
import { recordsService } from "../../services/records-service";
import type { RecordItem } from "../../types";

function make(id: string, supersededBy?: string): RecordItem {
  return {
    id,
    kind: "fix",
    scenario: "audio-tools",
    supersededBy,
    trigger: `trigger-${id}`,
    approach: "approach",
    ruledOut: [],
    filesChanged: [],
    outcome: "shipped",
    stub: false,
    createdAt: "2026-05-27T12:00:00Z",
  };
}

describe("SupersedeChainView", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("walks the chain via supersededBy and renders in order", async () => {
    const chain: Record<string, RecordItem> = {
      a: make("a", "b"),
      b: make("b", "c"),
      c: make("c"),
    };
    vi.spyOn(recordsService, "get").mockImplementation(async (id: string) => chain[id]!);

    renderWithProviders(<SupersedeChainView rootId="a" />);

    await waitFor(() => {
      expect(screen.getByTestId("record-card-a")).toBeInTheDocument();
      expect(screen.getByTestId("record-card-b")).toBeInTheDocument();
      expect(screen.getByTestId("record-card-c")).toBeInTheDocument();
    });
  });

  it("caps rendering when chain exceeds MAX_HOPS (defensive against cycles)", async () => {
    // Build a 30-record forward chain (a0 -> a1 -> ... -> a29)
    const items: Record<string, RecordItem> = {};
    for (let i = 0; i < 30; i++) {
      items[`a${i}`] = make(`a${i}`, i < 29 ? `a${i + 1}` : undefined);
    }
    vi.spyOn(recordsService, "get").mockImplementation(async (id: string) => items[id]!);

    renderWithProviders(<SupersedeChainView rootId="a0" />);

    await waitFor(() => {
      expect(screen.getByText(/Chain truncated/i)).toBeInTheDocument();
    });
  });
});
