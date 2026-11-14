import type { UseQueryResult } from "@tanstack/react-query";
import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { strings } from "../consts/strings";
import { renderWithProviders } from "../test-utils";
import { QueryView } from "./QueryView";

afterEach(() => cleanup());

const DATA = "the last good read";

function query(state: Partial<UseQueryResult<string>>): UseQueryResult<string> {
  return { isPending: false, isLoadingError: false, isRefetchError: false, error: null, data: DATA, dataUpdatedAt: Date.UTC(2026, 10, 11, 9, 30), refetch: () => Promise.resolve(), ...state } as UseQueryResult<string>;
}

describe("QueryView", () => {
  it("keeps an earlier read on screen and marks it not current when a refresh fails", () => {
    renderWithProviders(<QueryView query={query({ isRefetchError: true, error: new TypeError("Failed to fetch") })}>{(data) => <p>{data}</p>}</QueryView>);
    expect(screen.getByText(strings.errors.staleTitle)).toBeInTheDocument();
    expect(screen.getByText(DATA)).toBeInTheDocument();
  });

  it("replaces the page with the failure when there is nothing to show yet", () => {
    renderWithProviders(<QueryView query={query({ isLoadingError: true, data: undefined, error: new TypeError("Failed to fetch") })}>{(data) => <p>{data}</p>}</QueryView>);
    expect(screen.getByText(strings.errors.unreachableTitle)).toBeInTheDocument();
    expect(screen.queryByText(DATA)).not.toBeInTheDocument();
  });
});
