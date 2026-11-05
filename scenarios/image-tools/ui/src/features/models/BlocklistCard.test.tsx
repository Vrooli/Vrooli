/**
 * BlocklistCard tests — the read-only license-blocklist surface.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { makeBlocklistEntry, makeListBlocklistResponse } from "./mocks/factories";
import { makeModelsMocks } from "./mocks/models";

vi.mock("../../api/models", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/models")>();
  return { ...actual, ...makeModelsMocks() };
});

import { BlocklistCard } from "./BlocklistCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("BlocklistCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state when no models are blocked", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listBlocklist).mockResolvedValueOnce(makeListBlocklistResponse());

    renderWithProviders(<BlocklistCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.blocklist.empty)).toBeInTheDocument();
    });
  });

  it("lists blocked entries and shows the ONNX-export warning when flagged", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listBlocklist).mockResolvedValueOnce(
      makeListBlocklistResponse({
        entries: [
          makeBlocklistEntry({ id: "blocked-a", exportingOnnxRemovesRestriction: true }),
          makeBlocklistEntry({ id: "blocked-b", exportingOnnxRemovesRestriction: false }),
        ],
      }),
    );

    renderWithProviders(<BlocklistCard />);
    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.models.blocklist.entry)).toHaveLength(2);
    });
    // Only the flagged entry shows the export-trap warning.
    expect(screen.getAllByTestId(selectors.models.blocklist.onnxWarning)).toHaveLength(1);
  });

  it("renders the error state when listBlocklist rejects", async () => {
    const { modelsClient } = await import("../../api/models");
    vi.mocked(modelsClient.listBlocklist).mockRejectedValueOnce(new Error("down"));

    renderWithProviders(<BlocklistCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.models.blocklist.error)).toBeInTheDocument();
    });
  });
});
