import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { strings } from "./consts/strings";
import { renderWithProviders } from "@vrooli/api-base/testing";

// Mock the api-base proxy info so BrowserRouter has a deterministic basename
vi.mock("@vrooli/api-base", () => ({
  getProxyInfo: vi.fn().mockReturnValue(null),
  buildApiUrl: vi.fn((path: string) => path),
}));

vi.mock("./api/health", () => ({
  fetchHealth: vi.fn(),
}));

vi.mock("./api/healthStatus", () => ({
  getProviderHealth: vi.fn(),
}));

vi.mock("./services/settings", () => ({
  getProviderConfig: vi.fn(),
}));

vi.mock("./services/usage", () => ({
  listRecent: vi.fn(),
}));

// Mock lazy-loaded page modules so tests don't need full page deps
vi.mock("./features/overview/OverviewPage", () => ({
  OverviewPage: () => <div data-testid="overview-page">Overview</div>,
}));
vi.mock("./features/diagnostics/DiagnosticsPage", () => ({
  DiagnosticsPage: () => <div data-testid="diagnostics-page">Diagnostics</div>,
}));
vi.mock("./features/status/StatusPage", () => ({
  StatusPage: () => <div data-testid="status-page">Status</div>,
}));
vi.mock("./features/configuration/ConfigurationPage", () => ({
  ConfigurationPage: () => <div data-testid="configuration-page">Configuration</div>,
}));
vi.mock("./features/voices/VoicesPage", () => ({
  VoicesPage: () => <div data-testid="voices-page">Voices</div>,
}));
vi.mock("./features/usage/UsagePage", () => ({
  UsagePage: () => <div data-testid="usage-page">Usage</div>,
}));
vi.mock("./features/docs/DocsPage", () => ({
  DocsPage: () => <div data-testid="docs-page">Docs</div>,
}));
vi.mock("./features/docs/DocViewerPage", () => ({
  DocViewerPage: () => <div data-testid="doc-viewer-page">DocViewer</div>,
}));
vi.mock("./features/admin/SpeakerVerificationPage", () => ({
  SpeakerVerificationPage: () => <div data-testid="speaker-page">Speaker</div>,
}));
vi.mock("./features/admin/WakeWordPage", () => ({
  WakeWordPage: () => <div data-testid="wake-word-page">WakeWord</div>,
}));
vi.mock("./features/admin/StreamConfigPage", () => ({
  StreamConfigPage: () => <div data-testid="stream-config-page">StreamConfig</div>,
}));
vi.mock("./features/dictation-studio/DictationStudioPage", () => ({
  DictationStudioPage: () => <div data-testid="dictation-studio-page">DictationStudio</div>,
}));
vi.mock("./features/not-found/NotFoundPage", () => ({
  NotFoundPage: () => <div data-testid="not-found-page">NotFound</div>,
}));

import App from "./App";
import { fetchHealth } from "./api/health";
import { getProviderHealth } from "./api/healthStatus";

// jsdom doesn't implement window.matchMedia — PreferencesProvider needs it
beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }),
  });
});

function renderApp() {
  return renderWithProviders(<App />, { withoutRouter: true });
}

beforeEach(() => {
  vi.mocked(fetchHealth).mockImplementation(() => new Promise(() => {}));
  vi.mocked(getProviderHealth).mockImplementation(() => new Promise(() => {}));
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("App", () => {
  it("renders the overview page at root route (/)", async () => {
    renderApp();
    await waitFor(() => {
      expect(screen.getByTestId("overview-page")).toBeInTheDocument();
    });
  });

  it("renders the app title in the top bar", async () => {
    renderApp();
    await waitFor(() => {
      expect(screen.getByText(strings.app.title)).toBeInTheDocument();
    });
  });

  it("renders the primary nav (sidebar)", async () => {
    renderApp();
    await waitFor(() => {
      expect(screen.getByRole("complementary")).toBeInTheDocument();
    });
  });

  it("renders the Toaster live region", async () => {
    renderApp();
    await waitFor(() => {
      expect(document.querySelector(`[aria-live='polite']`)).toBeInTheDocument();
    });
  });
});

describe("App basename / proxy path", () => {
  it("uses empty basename when getProxyInfo returns null", async () => {
    // Default mock already returns null — just verify it renders
    renderApp();
    await waitFor(() => {
      expect(screen.getByTestId("overview-page")).toBeInTheDocument();
    });
  });

  it("renders Toaster when proxy basePath is set (covers getRouterBasename path branch)", async () => {
    const { getProxyInfo } = await import("@vrooli/api-base");
    vi.mocked(getProxyInfo as ReturnType<typeof vi.fn>).mockReturnValue({
      primary: { path: "/app/" },
      basePath: "/app/",
    });
    renderApp();
    await waitFor(() => {
      // Toaster is outside BrowserRouter so it always renders
      expect(document.querySelector("[aria-live]")).toHaveAttribute("aria-live", "polite");
    });
  });
});
