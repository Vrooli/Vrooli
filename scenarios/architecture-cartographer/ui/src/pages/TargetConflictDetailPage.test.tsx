import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";

vi.mock("../api/conflicts", () => ({
  conflictsClient: {
    listConflicts: vi.fn().mockResolvedValue({ conflicts: [], nextPageToken: "" }),
    getConflict: vi.fn().mockResolvedValue({ conflict: undefined }),
    detectConflicts: vi.fn(),
    validateConflicts: vi.fn(),
  },
}));

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TargetConflictDetailPage } from "./TargetConflictDetailPage";

afterEach(() => cleanup());

describe("TargetConflictDetailPage", () => {
  it("renders the detail page with a back link", async () => {
    renderWithProviders(
        <Routes>
          <Route
            path="/targets/:encodedPath/conflicts/:conflictId"
            element={<TargetConflictDetailPage />}
          />
        </Routes>,
      { routerEntries: ["/targets/demo/conflicts/c-1"] }
    );

    await waitFor(() =>
      expect(screen.getByTestId(selectors.pages.targetConflictDetail)).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.features.conflicts.detail.backLink)).toHaveAttribute(
      "href",
      "/targets/demo/conflicts",
    );
  });
});
