/**
 * App tests — smoke only.
 *
 * `App` is a tiny composition of `<Providers>` + `<AppRouter>`. Per-route
 * behaviour lives in `app/routes.test.tsx`, shell wiring in
 * `layout/AppShell.test.tsx`, theme persistence in
 * `theme/ThemeProvider.test.tsx`.
 *
 * One case uses `TestAppRouter` directly (the memory-router twin of the
 * production router); the `<App />` case mounts the real default `createBrowser
 * Router` at jsdom's default location ("/") wrapped only in the QueryClient +
 * i18n providers `main.tsx` owns — so the actual `App.tsx` composition runs.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { I18nextProvider } from "react-i18next";

import { renderWithProviders } from "./test-utils/renderWithProviders";
import { Providers } from "./app/providers";
import { TestAppRouter } from "./app/routes";
import { selectors } from "./consts/selectors";
import { i18n } from "./i18n";
import { makeListJobsResponse } from "./features/jobs/mocks/factories";

// Hoisted so the mock factories (which vitest lifts above the imports) can
// reference them without a temporal-dead-zone error.
const mocks = vi.hoisted(async () => {
  const { makeHealthResponse } = await import("./test-utils/factories");
  return { health: makeHealthResponse() };
});

vi.mock("./api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./api/health")>();
  return { ...actual, fetchHealth: vi.fn().mockResolvedValue((await mocks).health) };
});

vi.mock("./api/jobs", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./api/jobs")>();
  return {
    ...actual,
    jobsClient: { listJobs: vi.fn(), cancelJob: vi.fn(), watchJob: vi.fn() },
  };
});

import App from "./App";
import { jobsClient } from "./api/jobs";

describe("App composition", () => {
  beforeEach(() => {
    vi.mocked(jobsClient.listJobs).mockResolvedValue(makeListJobsResponse({ jobs: [] }));
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the shell title (smoke: providers + routes wire up)", async () => {
    renderWithProviders(
      <Providers>
        <TestAppRouter initialEntries={["/"]} />
      </Providers>,
      { withoutRouter: true },
    );
    expect(screen.getByTestId(selectors.app.title)).toBeInTheDocument();
    // Let the Home jobs query settle so its resolution doesn't bleed into the
    // next test as an unhandled rejection / console error.
    await waitFor(() => {
      expect(screen.getByTestId(selectors.pages.home)).toBeInTheDocument();
    });
  });
});

describe("App (real default router at /)", () => {
  beforeEach(() => {
    vi.mocked(jobsClient.listJobs).mockResolvedValue(makeListJobsResponse({ jobs: [] }));
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("mounts <App /> with its own providers + browser router and renders the shell", async () => {
    // <App /> wires Theme + Settings + the production createBrowserRouter; the
    // QueryClient and i18n providers normally live in main.tsx, so supply them
    // here. jsdom's default location is "/", which resolves to the Home route.
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <App />
        </I18nextProvider>
      </QueryClientProvider>,
    );
    await waitFor(() => {
      expect(screen.getByTestId(selectors.app.title)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.layout.shell)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.pages.home)).toBeInTheDocument();
  });
});
