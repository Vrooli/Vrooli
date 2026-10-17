/**
 * Self-test for renderWithProviders.
 *
 * This helper is the load-bearing test infrastructure for every scenario UI
 * suite in the fleet, so its contracts are pinned here — in the package that
 * owns the code — rather than in each consumer. If a refactor ever flips
 * `defaultOptions.queries.retry` from `false` to the React Query default
 * (3 retries with exponential backoff), every UI test that exercises an error
 * path silently weakens: it would still pass, but only after waiting through
 * the retry window.
 *
 * The contracts that must not regress:
 *
 *   1. The default QueryClient disables query retries
 *   2. The default QueryClient disables mutation retries
 *   3. The returned `queryClient` is the one the rendered tree used
 *   4. A custom queryClient option is honored (cache seeding flows)
 *   5. The router defaults to "/" and honors initialEntries / the route alias
 *   6. i18n defaults to *this package's* instance unless the caller supplies one
 *   7. Scenario-owned providers compose around the shared defaults
 *
 * On contract 6 — read this before "simplifying" it. api-base deliberately
 * defaults to an i18next instance it creates itself, pinned to `cimode` with
 * empty resources. A test that renders `t("some.key")` and asserts the key
 * path echoes back CANNOT distinguish that default from a scenario's own
 * singleton, because cimode echoes the key either way. Asserting instance
 * identity is the only form of this test that can fail, so that is the form
 * kept here.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { QueryClient, useMutation, useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { createInstance } from "i18next";
import { useLocation } from "react-router-dom";

import {
  configureTestProviders,
  createTestQueryClient,
  renderWithProviders,
} from "./renderWithProviders";

afterEach(() => {
  cleanup();
  configureTestProviders(undefined);
});

describe("renderWithProviders default QueryClient", () => {
  it("disables query retries", () => {
    const { queryClient } = renderWithProviders(<div />);
    expect(queryClient.getDefaultOptions().queries?.retry).toBe(false);
  });

  it("disables mutation retries", () => {
    const { queryClient } = renderWithProviders(<div />);
    expect(queryClient.getDefaultOptions().mutations?.retry).toBe(false);
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
      expect(screen.getByTestId("probe-state").textContent).toBe("error");
    });
  });

  it("fails a mutation immediately rather than retrying on rejection", async () => {
    const Probe = () => {
      const m = useMutation({ mutationFn: () => Promise.reject(new Error("boom")) });
      return (
        <button data-testid="probe-button" type="button" onClick={() => m.mutate()}>
          {m.status}
        </button>
      );
    };

    renderWithProviders(<Probe />);
    screen.getByTestId("probe-button").click();
    await waitFor(() => {
      expect(screen.getByTestId("probe-button").textContent).toBe("error");
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
      expect(screen.getByTestId("seeded").textContent).toBe("hello");
    });
  });
});

describe("renderWithProviders router", () => {
  // Read the location from the router, not `window.location`. MemoryRouter
  // keeps its history in memory and never touches the jsdom URL, so a
  // `window.location.pathname` assertion here reads "/" no matter what was
  // passed and can only ever fail.
  const RouteProbe = () => {
    const location = useLocation();
    return <output data-testid="route">{location.pathname}</output>;
  };

  it("renders at the root path by default", () => {
    renderWithProviders(<RouteProbe />);
    expect(screen.getByTestId("route").textContent).toBe("/");
  });

  it("honors initialEntries and the legacy single-route alias", () => {
    const queryClient = createTestQueryClient();

    renderWithProviders(<RouteProbe />, { queryClient, initialEntries: ["/legacy"] });
    expect(screen.getByTestId("route").textContent).toBe("/legacy");

    cleanup();

    renderWithProviders(<RouteProbe />, { route: "/aliased" });
    expect(screen.getByTestId("route").textContent).toBe("/aliased");
  });
});

describe("renderWithProviders i18n wiring", () => {
  it("uses its own default instance unless the caller supplies one", () => {
    const supplied = createInstance();
    void supplied.init({
      initImmediate: false,
      lng: "cimode",
      fallbackLng: false,
      resources: {},
      interpolation: { escapeValue: false },
    });

    const Probe = () => {
      const { i18n } = useTranslation();
      return <span data-testid="who">{i18n === supplied ? "supplied" : "default"}</span>;
    };

    renderWithProviders(<Probe />);
    expect(screen.getByTestId("who").textContent).toBe("default");

    cleanup();

    renderWithProviders(<Probe />, { i18n: supplied });
    expect(screen.getByTestId("who").textContent).toBe("supplied");
  });
});

describe("renderWithProviders scenario-owned providers", () => {
  it("composes scenario-owned providers around the shared defaults", () => {
    configureTestProviders((children) => (
      <section data-testid="scenario-provider">{children}</section>
    ));

    renderWithProviders(<span>child</span>);

    expect(
      screen.getByTestId("scenario-provider").contains(screen.getByText("child")),
    ).toBe(true);
  });
});
