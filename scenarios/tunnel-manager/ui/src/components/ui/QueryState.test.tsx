/**
 * QueryState tests — the shared loading / error / empty / data fallbacks every
 * surface routes its query results through.
 */
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { I18nextProvider } from "react-i18next";

import { i18n, setLocale } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { QueryState } from "./QueryState";

const renderState = (ui: React.ReactElement) =>
  render(<I18nextProvider i18n={i18n}>{ui}</I18nextProvider>);

describe("QueryState", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => cleanup());

  it("renders the loading fallback while loading", () => {
    renderState(
      <QueryState isLoading error={null}>
        <div data-testid="child" />
      </QueryState>,
    );
    expect(screen.getByTestId(selectors.queryState.loading)).toBeInTheDocument();
    expect(screen.queryByTestId("child")).not.toBeInTheDocument();
  });

  it("renders the error fallback (normalised) when an error is present", () => {
    renderState(
      <QueryState isLoading={false} error={new Error("boom")}>
        <div data-testid="child" />
      </QueryState>,
    );
    expect(screen.getByTestId(selectors.queryState.error)).toBeInTheDocument();
  });

  it("prefers an explicit error label over the normalised message", () => {
    renderState(
      <QueryState isLoading={false} error={new Error("boom")} errorLabel="custom error">
        <div data-testid="child" />
      </QueryState>,
    );
    expect(screen.getByTestId(selectors.queryState.error)).toHaveTextContent("custom error");
  });

  it("renders the empty fallback when isEmpty is set", () => {
    renderState(
      <QueryState isLoading={false} error={null} isEmpty emptyLabel="nothing here">
        <div data-testid="child" />
      </QueryState>,
    );
    expect(screen.getByTestId(selectors.queryState.empty)).toHaveTextContent("nothing here");
  });

  it("renders children when data is present and non-empty", () => {
    renderState(
      <QueryState isLoading={false} error={null}>
        <div data-testid="child" />
      </QueryState>,
    );
    expect(screen.getByTestId("child")).toBeInTheDocument();
  });
});
