/**
 * Self-test for renderWithProviders.
 *
 * The render helper is the load-bearing test infrastructure for every
 * component test in this scenario. If a refactor ever flips
 * `defaultOptions.queries.retry` from `false` to the React Query
 * default (3 retries with exponential backoff), every UI test that
 * exercises an error path silently weakens — the test would still pass,
 * but only after waiting through the retry window. This file pins the
 * five contracts that must not regress:
 *
 *   1. The default QueryClient disables query retries
 *   2. The default QueryClient disables mutation retries
 *   3. The returned `queryClient` is the one the rendered tree used
 *   4. A custom queryClient prop is honored (cache seeding flows)
 *   5. I18nextProvider is wired to the same singleton App.tsx uses
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { QueryClient, useMutation, useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";

import { renderWithProviders } from "@vrooli/api-base/testing";
import { strings } from "../consts/strings";

afterEach(() => cleanup());

describe("renderWithProviders default QueryClient", () => {
  it("disables query retries", () => {
    const { queryClient } = renderWithProviders(<div />);
    const defaults = queryClient.getDefaultOptions();
    expect(defaults.queries?.retry).toBe(false);
  });

  it("disables mutation retries", () => {
    const { queryClient } = renderWithProviders(<div />);
    const defaults = queryClient.getDefaultOptions();
    expect(defaults.mutations?.retry).toBe(false);
  });

  it("fails a query immediately rather than retrying on rejection", async () => {
    const Probe = () => {
      const q = useQuery({
        queryKey: ["probe"],
        queryFn: () => Promise.reject(new Error("boom")),
      });
      return <span data-testid="probe-state">{q.status}</span>;
    };

    renderWithProviders(<Probe />);
    await waitFor(() => {
      expect(screen.getByTestId("probe-state")).toHaveTextContent("error");
    });
  });

  it("fails a mutation immediately rather than retrying on rejection", async () => {
    const Probe = () => {
      const m = useMutation({ mutationFn: () => Promise.reject(new Error("boom")) });
      return (
        <button
          data-testid="probe-button"
          type="button"
          onClick={() => m.mutate()}
        >
          {m.status}
        </button>
      );
    };

    renderWithProviders(<Probe />);
    screen.getByTestId("probe-button").click();
    await waitFor(() => {
      expect(screen.getByTestId("probe-button")).toHaveTextContent("error");
    });
  });
});

describe("renderWithProviders QueryClient identity", () => {
  it("returns the QueryClient that the rendered tree used", async () => {
    const Probe = () => {
      useQuery({ queryKey: ["identity"], queryFn: () => Promise.resolve(42) });
      return <span data-testid="probe">ok</span>;
    };

    const { queryClient } = renderWithProviders(<Probe />);
    await waitFor(() => {
      expect(queryClient.getQueryData(["identity"])).toBe(42);
    });
  });

  it("honors a custom queryClient option", async () => {
    const seeded = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    seeded.setQueryData(["seeded"], "hello");

    const Probe = () => {
      const q = useQuery({
        queryKey: ["seeded"],
        // queryFn never runs because seeded data is fresh.
        queryFn: () => Promise.resolve("network"),
      });
      return <span data-testid="seeded">{String(q.data)}</span>;
    };

    const { queryClient } = renderWithProviders(<Probe />, { queryClient: seeded });
    expect(queryClient).toBe(seeded);
    await waitFor(() => {
      expect(screen.getByTestId("seeded")).toHaveTextContent("hello");
    });
  });
});

describe("renderWithProviders I18nextProvider wiring", () => {
  it("binds I18nextProvider to the singleton (cimode echoes the key path)", () => {
    const Probe = () => {
      const { t } = useTranslation();
      return <span data-testid="probe-key">{t(strings.app.title)}</span>;
    };

    renderWithProviders(<Probe />);
    // test-setup.ts puts the singleton in cimode; if renderWithProviders
    // accidentally constructed its own i18n instance, this would render
    // the translated copy ("Source Ledger" or similar)
    // rather than the literal key path.
    expect(screen.getByTestId("probe-key")).toHaveTextContent("app.title");
  });
});
