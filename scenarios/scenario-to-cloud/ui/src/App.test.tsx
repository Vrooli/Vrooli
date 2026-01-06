import "@testing-library/jest-dom";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import App from "./App";
import { selectors } from "./consts/selectors";

const SAVED_DEPLOYMENT_KEY = "scenario-to-cloud:deployment";

function jsonResponse(payload: unknown, status = 200) {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { "Content-Type": "application/json" }
  });
}

function seedSavedDeployment() {
  const manifest = {
    version: "1.0.0",
    target: {
      type: "vps",
      vps: {
        host: "203.0.113.10",
      },
    },
    scenario: {
      id: "demo-scenario",
    },
    dependencies: {
      scenarios: ["demo-scenario"],
      resources: [],
    },
    bundle: {
      include_packages: true,
      include_autoheal: true,
    },
    ports: {
      ui: 3000,
      api: 3001,
      ws: 3002,
    },
    edge: {
      domain: "example.com",
      caddy: {
        enabled: true,
        email: "",
      },
    },
  };

  localStorage.setItem(
    SAVED_DEPLOYMENT_KEY,
    JSON.stringify({
      manifestJson: JSON.stringify(manifest, null, 2),
      currentStep: 0,
      timestamp: Date.now(),
    })
  );
}

function renderApp() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  );
}

describe("Dashboard", () => {
  beforeEach(() => {
    window.history.replaceState(null, "", "#dashboard");
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/health")) {
        return jsonResponse({ status: "healthy", service: "Scenario To Cloud API", timestamp: "2025-01-01T00:00:00Z" });
      }
      return jsonResponse({ error: { message: "not found" } }, 404);
    }));
    localStorage.clear();
  });

  test("shows dashboard on initial load", async () => {
    renderApp();

    // Dashboard should be visible with "Start New" button
    expect(await screen.findByTestId(selectors.dashboard.startNewButton)).toBeInTheDocument();
    expect(screen.getByText("Deploy Scenarios to the Cloud")).toBeInTheDocument();
  });

  test("navigates to wizard when clicking Start New", async () => {
    renderApp();

    const startNewButton = await screen.findByTestId(selectors.dashboard.startNewButton);
    await userEvent.click(startNewButton);

    // Wizard should now be visible
    expect(await screen.findByTestId(selectors.wizard.container)).toBeInTheDocument();
  });
});

describe("Deployment Wizard", () => {
  beforeEach(() => {
    window.history.replaceState(null, "", "#dashboard");
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/health")) {
        return jsonResponse({ status: "healthy", service: "Scenario To Cloud API", timestamp: "2025-01-01T00:00:00Z" });
      }
      if (url.includes("/manifest/validate")) {
        return jsonResponse({ valid: true, issues: [], timestamp: "2025-01-01T00:00:00Z" });
      }
      if (url.includes("/secrets/")) {
        return jsonResponse({
          secrets: {
            bundle_secrets: [],
            summary: {
              total_secrets: 0,
              infrastructure: 0,
              per_install_generated: 0,
              user_prompt: 0,
              remote_fetch: 0,
            }
          }
        });
      }
      if (url.includes("/plan")) {
        return jsonResponse({
          plan: [
            { id: "1", title: "Upload bundle", description: "Copy bundle to server" },
            { id: "2", title: "Run setup", description: "Configure environment" },
          ],
          timestamp: "2025-01-01T00:00:00Z"
        });
      }
      if (url.includes("/bundle/build")) {
        return jsonResponse({ artifact: { path: "/tmp/mini.tar.gz", sha256: "abc123", size_bytes: 123456 }, timestamp: "2025-01-01T00:00:00Z" });
      }
      return jsonResponse({ error: { message: "not found" } }, 404);
    }));
    localStorage.clear();
    seedSavedDeployment();
  });

  async function navigateToWizard() {
    renderApp();
    const startNewButton = await screen.findByTestId(selectors.dashboard.startNewButton);
    await userEvent.click(startNewButton);
    await screen.findByTestId(selectors.wizard.container);
  }

  test("validates a manifest and shows result", async () => {
    // [REQ:STC-P0-001] cloud manifest validation is exposed through the UI
    await navigateToWizard();

    // Validation runs automatically on step entry
    // Wait for validation to complete and show result
    expect(await screen.findByText("Manifest is valid")).toBeInTheDocument();
  });

  test("advances to secrets step after validation", async () => {
    await navigateToWizard();

    // Wait for validation to complete
    await screen.findByText("Manifest is valid");

    // Navigate to secrets step
    const nextButton = screen.getByTestId(selectors.wizard.nextButton);
    await waitFor(() => expect(nextButton).toBeEnabled());
    await userEvent.click(nextButton);

    // Secrets step should render
    expect(await screen.findByRole("heading", { name: "Secrets" })).toBeInTheDocument();
  });

  test("builds bundle and shows artifact details", async () => {
    await navigateToWizard();

    // Wait for validation to complete
    await screen.findByText("Manifest is valid");

    // Navigate to secrets step
    const nextButton = screen.getByTestId(selectors.wizard.nextButton);
    await waitFor(() => expect(nextButton).toBeEnabled());
    await userEvent.click(nextButton);
    await screen.findByRole("heading", { name: "Secrets" });

    // Navigate to build step
    const nextFromSecrets = screen.getByTestId(selectors.wizard.nextButton);
    await waitFor(() => expect(nextFromSecrets).toBeEnabled());
    await userEvent.click(nextFromSecrets);

    await screen.findByTestId(selectors.manifest.bundleBuildButton);

    // Click build button
    await userEvent.click(screen.getByTestId(selectors.manifest.bundleBuildButton));

    // Wait for bundle result
    await waitFor(async () => {
      expect(await screen.findByTestId(selectors.manifest.bundleBuildResult)).toHaveTextContent(
        "/tmp/mini.tar.gz"
      );
    });
  });

  test("can navigate back to dashboard", async () => {
    await navigateToWizard();

    // Click dashboard button
    await userEvent.click(screen.getByTestId(selectors.wizard.dashboardButton));

    // Should be back on dashboard
    expect(await screen.findByTestId(selectors.dashboard.startNewButton)).toBeInTheDocument();
  });
});
