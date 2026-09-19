import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

vi.mock("../../api/measures", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/measures")>();
  return { ...actual, countFailed: vi.fn(), countCoverage: vi.fn() };
});

import { MeasuresCard } from "./MeasuresCard";
import { countCoverage, countFailed, TimeWindowToken } from "../../api/measures";

const mockFailed = vi.mocked(countFailed);
const mockCoverage = vi.mocked(countCoverage);

describe("MeasuresCard", () => {
  // Counts are interpolated into the labels, so opt into a real locale
  // (default test locale is cimode, which drops `{{count}}`).
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the failed and passed counts for the default (this-week) window", async () => {
    mockFailed.mockResolvedValue(3n);
    mockCoverage.mockResolvedValue(7n);

    renderWithProviders(<MeasuresCard />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.measures.failedValue)).toBeInTheDocument(),
    );

    // Counts are response data, asserted via the rendered count, not UI copy.
    expect(screen.getByTestId(selectors.measures.failedValue)).toHaveTextContent("3");
    expect(screen.getByTestId(selectors.measures.passedValue)).toHaveTextContent("7");

    // Defaults to THIS_WEEK.
    expect(mockFailed).toHaveBeenCalledWith(TimeWindowToken.THIS_WEEK);
    expect(mockCoverage).toHaveBeenCalledWith(TimeWindowToken.THIS_WEEK);
  });

  it("refetches both counts for the selected window", async () => {
    mockFailed.mockResolvedValue(1n);
    mockCoverage.mockResolvedValue(2n);

    renderWithProviders(<MeasuresCard />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.measures.failedValue)).toBeInTheDocument(),
    );

    mockFailed.mockResolvedValue(5n);
    mockCoverage.mockResolvedValue(9n);

    await userEvent.selectOptions(
      screen.getByTestId(selectors.measures.windowSelect),
      String(TimeWindowToken.LAST_30D),
    );

    await waitFor(() =>
      expect(screen.getByTestId(selectors.measures.failedValue)).toHaveTextContent("5"),
    );
    expect(screen.getByTestId(selectors.measures.passedValue)).toHaveTextContent("9");
    expect(mockFailed).toHaveBeenLastCalledWith(TimeWindowToken.LAST_30D);
    expect(mockCoverage).toHaveBeenLastCalledWith(TimeWindowToken.LAST_30D);
  });

  it("shows the error state when a count RPC fails", async () => {
    mockFailed.mockRejectedValue(new Error("boom"));
    mockCoverage.mockResolvedValue(0n);

    renderWithProviders(<MeasuresCard />);

    await waitFor(() => expect(screen.getByTestId(selectors.measures.error)).toBeInTheDocument());
  });
});
