import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

vi.mock("../api/graph", () => ({
  graphClient: {
    listGraphSnapshots: vi.fn().mockResolvedValue({ snapshots: [], nextPageToken: "" }),
    extractGraph: vi.fn(),
  },
}));

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { NewTargetPage } from "./NewTargetPage";

afterEach(() => cleanup());

describe("NewTargetPage", () => {
  it("renders the page and the NewTargetForm inside it", () => {
    renderWithProviders(<NewTargetPage />);
    expect(screen.getByTestId(selectors.pages.newTarget)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.features.targets.newForm.root)).toBeInTheDocument();
  });
});
