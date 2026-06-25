/**
 * AsyncBoundary tests — the three non-happy states (loading / error / empty)
 * each render their own role + test id, and children render only when settled.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { I18nextProvider } from "react-i18next";

import { AsyncBoundary } from "./AsyncBoundary";
import { selectors } from "../consts/selectors";
import { i18n } from "../i18n";
import { makeApiError } from "../api/client";

const wrap = (ui: React.ReactElement) =>
  render(<I18nextProvider i18n={i18n}>{ui}</I18nextProvider>);

describe("AsyncBoundary", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the loading state with a status role", () => {
    wrap(
      <AsyncBoundary isLoading error={null} testIdPrefix="x">
        <span>child</span>
      </AsyncBoundary>,
    );
    expect(screen.getByTestId(`x-${selectors.asyncSuffix.loading}`)).toHaveAttribute("role", "status");
  });

  it("renders a translated error with an alert role", () => {
    wrap(
      <AsyncBoundary isLoading={false} error={makeApiError("not_found", "missing", 404)} testIdPrefix="x">
        <span>child</span>
      </AsyncBoundary>,
    );
    expect(screen.getByTestId(`x-${selectors.asyncSuffix.error}`)).toHaveAttribute("role", "alert");
  });

  it("renders the empty state when isEmpty is set", () => {
    wrap(
      <AsyncBoundary isLoading={false} error={null} isEmpty testIdPrefix="x">
        <span>child</span>
      </AsyncBoundary>,
    );
    expect(screen.getByTestId(`x-${selectors.asyncSuffix.empty}`)).toBeInTheDocument();
  });

  it("renders children when settled and non-empty", () => {
    wrap(
      <AsyncBoundary isLoading={false} error={null} testIdPrefix="x">
        <span data-testid="child" />
      </AsyncBoundary>,
    );
    expect(screen.getByTestId("child")).toBeInTheDocument();
  });
});
