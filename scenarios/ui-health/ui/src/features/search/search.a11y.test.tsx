import { describe, it, vi, beforeEach } from "vitest";
import userEvent from "@testing-library/user-event";
import { screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { selectors } from "../../consts/selectors";
import type { SearchHit, SearchResults } from "../../api/search";

const makeHit = (overrides: Partial<SearchHit> = {}): SearchHit => ({
  scenario: "ui-health",
  slot: "DashboardCard",
  kind: "component",
  displayName: "DashboardCard",
  description: "A card on the dashboard.",
  filePath: "ui/src/components/DashboardCard.tsx",
  score: 0.87,
  provenance: "custom",
  library: "",
  componentName: "",
  ...overrides,
});

vi.mock("../../api/search", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/search")>();
  return {
    ...actual,
    searchSurfaces: vi.fn(
      (): Promise<SearchResults> =>
        Promise.resolve({
          hits: [makeHit(), makeHit({ slot: "Foo", kind: "page" })],
          modeUsed: "text",
        }),
    ),
  };
});

import { SearchPage } from "./SearchPage";

beforeEach(() => {
  // no-op
});

describe("SearchPage accessibility", () => {
  it("has no axe violations in the empty state", async () => {
    const { container } = renderWithProviders(<SearchPage />);
    await expectNoA11yViolations(container);
  });

  it("has no axe violations once results are rendered", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<SearchPage />);
    await user.type(screen.getByTestId(selectors.search.input), "card");
    await screen.findByTestId(selectors.search.resultsList);
    await expectNoA11yViolations(container);
  });
});
