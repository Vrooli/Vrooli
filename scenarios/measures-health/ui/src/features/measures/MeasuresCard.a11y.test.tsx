/**
 * MeasuresCard accessibility regression test. The card owns its own query
 * state + window selector, so the axe wait and the count mocks live with the
 * feature rather than leaking into the app-level a11y suite.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

vi.mock("../../api/measures", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/measures")>();
  return { ...actual, countFailed: vi.fn(), countCoverage: vi.fn() };
});

import { MeasuresCard } from "./MeasuresCard";
import { countCoverage, countFailed } from "../../api/measures";

describe("MeasuresCard accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the loaded card without axe violations", async () => {
    vi.mocked(countFailed).mockResolvedValue(3n);
    vi.mocked(countCoverage).mockResolvedValue(7n);

    const { container } = renderWithProviders(<MeasuresCard />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.measures.failedValue)).toBeInTheDocument(),
    );
    await expectNoA11yViolations(container);
  });
});
