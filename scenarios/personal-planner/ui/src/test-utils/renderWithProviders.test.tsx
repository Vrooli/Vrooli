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
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { QueryClient, useMutation, useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";

import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "./index";
import { i18n, setLocale } from "../i18n";
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
    const user = userEvent.setup();
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
    await user.click(screen.getByRole("button", { name: "idle" }));
    await waitFor(() => {
      expect(screen.getByTestId("probe-button")).toHaveTextContent("error");
    });
  });
});

describe("renderWithProviders QueryClient identity", () => {
  it("cancels a signal-aware pending query when its view unmounts", async () => {
    const aborted = vi.fn();
    const query = vi.fn(({ signal }: { signal: AbortSignal }) => new Promise<string>((_resolve, reject) => {
      signal.addEventListener("abort", () => {
        aborted();
        reject(new DOMException("Cancelled", "AbortError"));
      }, { once: true });
    }));
    const Probe = () => {
      const result = useQuery({ queryKey: ["pending"], queryFn: query });
      return <output aria-label="query state">{result.status}</output>;
    };
    const view = renderWithProviders(<Probe />);
    await waitFor(() => expect(query).toHaveBeenCalledTimes(1));
    expect(screen.getByRole("status", { name: "query state" })).toHaveTextContent("pending");
    view.unmount();
    await waitFor(() => expect(aborted).toHaveBeenCalledTimes(1));
    expect(view.queryClient.getQueryData(["pending"])).toBeUndefined();
  });

  it("does not share cached values between independent renders", () => {
    const first = renderWithProviders(<div />);
    first.queryClient.setQueryData(["private"], "first render");
    const second = renderWithProviders(<div />);
    expect(second.queryClient).not.toBe(first.queryClient);
    expect(second.queryClient.getQueryData(["private"])).toBeUndefined();
    expect(first.queryClient.getQueryData(["private"])).toBe("first render");
  });
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
        staleTime: Infinity,
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
  it.each(["ja", "ar"] as const)("starts clean before selecting %s", async locale => {
    expect(i18n.language).toBe("cimode");
    expect(document.documentElement.dir).toBe("ltr");
    expect(window.localStorage.getItem("vrooli.locale")).toBeNull();
    await setLocale(locale);
    expect(i18n.language).toBe(locale);
    expect(window.localStorage.getItem("vrooli.locale")).toBe(locale);
    expect(document.documentElement.dir).toBe(locale === "ar" ? "rtl" : "ltr");
    // Leave changed state intentionally: the next test must receive clean setup.
  });

  it("renders the scenario's real locale, not an empty base-provider instance", async () => {
    await setLocale("ja");
    const Probe = () => {
      const { t } = useTranslation();
      return <button>{t(strings.health.refresh)}</button>;
    };
    renderWithProviders(<Probe />);
    expect(screen.getByRole("button", { name: i18n.t(strings.health.refresh) })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "health.refresh" })).not.toBeInTheDocument();
  });

  it("binds I18nextProvider to the singleton (cimode echoes the key path)", () => {
    const Probe = () => {
      const { t } = useTranslation();
      return <span data-testid="probe-key">{t(strings.app.title)}</span>;
    };

    renderWithProviders(<Probe />);
    // test-setup.ts puts the singleton in cimode; if renderWithProviders
    // accidentally constructed its own i18n instance, this would render
    // the translated copy ("Personal Planner" or similar)
    // rather than the literal key path.
    expect(screen.getByTestId("probe-key")).toHaveTextContent("app.title");
  });
});
