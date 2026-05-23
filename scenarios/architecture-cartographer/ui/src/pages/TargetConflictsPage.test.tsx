import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

vi.mock("../api/conflicts", () => ({
  conflictsClient: {
    listConflicts: vi.fn().mockResolvedValue({ conflicts: [], nextPageToken: "" }),
    detectConflicts: vi.fn(),
    validateConflicts: vi.fn(),
  },
}));

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TargetConflictsPage } from "./TargetConflictsPage";

afterEach(() => cleanup());

describe("TargetConflictsPage", () => {
  it("renders the conflicts page under the workspace route", async () => {
    renderWithProviders(
      <MemoryRouter
        initialEntries={["/targets/demo/conflicts"]}
        future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
      >
        <Routes>
          <Route path="/targets/:encodedPath/conflicts" element={<TargetConflictsPage />} />
        </Routes>
      </MemoryRouter>,
      { withoutRouter: true },
    );

    await waitFor(() =>
      expect(screen.getByTestId(selectors.pages.targetConflicts)).toBeInTheDocument(),
    );
  });
});
