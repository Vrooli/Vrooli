/**
 * Navigation spec component-tier conformance test (per
 * navigation-integrity-audit §5).
 *
 * For each `nav_*` affordance declared in `ui/flow/navigation.json`:
 *   - render the App under the affordance's host container context,
 *   - assert the element exists with the declared `test_id`,
 *   - click it,
 *   - assert the resulting URL is the destination route's path.
 *
 * Viewport-keyed: matches `(max-width: 767px)` get stubbed to true for
 * mobile-only presentations, false for desktop-only presentations.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import navigationSpec from "../../../flow/navigation.json";
import { makeApiMocks } from "../../test-utils";

vi.mock("../../api/golden", async (importOriginal) => {
  const { create } = await import("@bufbuild/protobuf");
  const { ListGoldensResponseSchema } = await import(
    "@vrooli/proto-types/development-toolchain-validator/v1/golden/golden_pb"
  );
  const actual = await importOriginal<typeof import("../../api/golden")>();
  return {
    ...actual,
    goldenClient: {
      listGoldens: vi.fn().mockResolvedValue(create(ListGoldensResponseSchema, { goldens: [] })),
      getGolden: vi.fn(),
      registerGolden: vi.fn(),
      regenerateGolden: vi.fn(),
      deleteGolden: vi.fn(),
    },
  };
});

vi.mock("../../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/health")>();
  return { ...actual, ...makeApiMocks() };
});

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { I18nextProvider } from "react-i18next";
import { MemoryRouter, useLocation } from "react-router-dom";
import { i18n } from "../../i18n";
import { AppRoutes } from "../../App";

interface Presentation {
  in: string;
  label: string;
  test_id: string;
  reachable_via: readonly string[];
}

interface Affordance {
  id: string;
  to: string;
  presentations: Presentation[];
}

interface Route {
  id: string;
  path: string;
}

const spec = navigationSpec as {
  routes: Route[];
  affordances: Affordance[];
};

const routeById = new Map<string, Route>(spec.routes.map((r) => [r.id, r]));

const NAV_AFFORDANCES = spec.affordances.filter((aff) =>
  aff.presentations.some((p) => p.in === "sidebar" || p.in === "mobile_bottom_nav"),
);

function stubMatchMedia(isMobile: boolean) {
  const mql = (q: string) =>
    ({
      matches: q.includes("max-width: 767px") ? isMobile : false,
      media: q,
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
      onchange: null,
      addListener: () => undefined,
      removeListener: () => undefined,
      dispatchEvent: () => false,
    }) as unknown as MediaQueryList;
  return vi.spyOn(window, "matchMedia").mockImplementation(mql);
}

function LocationProbe() {
  const loc = useLocation();
  return <div data-testid="location-probe" data-pathname={loc.pathname} />;
}

function renderApp(routerEntries: readonly string[] = ["/"]) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <I18nextProvider i18n={i18n}>
        <MemoryRouter initialEntries={[...routerEntries]}>
          <AppRoutes />
          <LocationProbe />
        </MemoryRouter>
      </I18nextProvider>
    </QueryClientProvider>,
  );
}

describe("navigation spec — affordance conformance", () => {
  let mqSpy: ReturnType<typeof stubMatchMedia> | null = null;

  beforeEach(() => {
    mqSpy = null;
  });

  afterEach(() => {
    cleanup();
    mqSpy?.mockRestore();
  });

  for (const aff of NAV_AFFORDANCES) {
    for (const presentation of aff.presentations) {
      if (presentation.in !== "sidebar" && presentation.in !== "mobile_bottom_nav") continue;
      const dest = routeById.get(aff.to);
      if (!dest) continue;
      const isMobilePres = presentation.in === "mobile_bottom_nav";

      it(`${aff.id} in ${presentation.in} → ${dest.path}`, async () => {
        mqSpy = stubMatchMedia(isMobilePres);
        renderApp();
        await waitFor(() => {
          expect(screen.getByTestId(presentation.test_id)).toBeInTheDocument();
        });
        await userEvent.click(screen.getByTestId(presentation.test_id));
        await waitFor(() => {
          expect(screen.getByTestId("location-probe").getAttribute("data-pathname")).toBe(
            dest.path,
          );
        });
      });
    }
  }
});
