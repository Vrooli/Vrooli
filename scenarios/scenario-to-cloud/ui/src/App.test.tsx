import "@testing-library/jest-dom";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import App from "./App";
import { selectors } from "./consts/selectors";

function jsonResponse(payload: unknown, status = 200) {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { "Content-Type": "application/json" }
  });
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
    vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/health")) {
        return jsonResponse({ status: "healthy", service: "Scenario To Cloud API", timestamp: "2025-01-01T00:00:00Z" });
      }
      if (url.includes("/manifest/validate")) {
        return jsonResponse({ valid: true, issues: [], timestamp: "2025-01-01T00:00:00Z" });
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

    // Navigate to validate step by clicking Continue
    await userEvent.click(screen.getByTestId(selectors.wizard.nextButton));

    // Validation runs automatically on step entry
    // Wait for validation to complete and show result
    await waitFor(async () => {
      expect(await screen.findByTestId(selectors.manifest.validateResult)).toHaveTextContent(
        "Manifest is Valid"
      );
    });
  });

  test("generates deployment plan", async () => {
    await navigateToWizard();

    // Navigate to validate step
    await userEvent.click(screen.getByTestId(selectors.wizard.nextButton));

    // Wait for validation to complete
    await screen.findByTestId(selectors.manifest.validateResult);

    // Navigate to plan step
    await userEvent.click(screen.getByTestId(selectors.wizard.nextButton));

    // Plan generates automatically on step entry
    // Wait for plan to generate
    await waitFor(async () => {
      expect(await screen.findByTestId(selectors.manifest.planResult)).toHaveTextContent(
        "Upload bundle"
      );
    });
  });

  test("builds bundle and shows artifact details", async () => {
    await navigateToWizard();

    // Navigate to validate step
    await userEvent.click(screen.getByTestId(selectors.wizard.nextButton));
    await screen.findByTestId(selectors.manifest.validateResult);

    // Navigate to plan step
    await userEvent.click(screen.getByTestId(selectors.wizard.nextButton));
    await screen.findByTestId(selectors.manifest.planResult);

    // Navigate to build step
    await userEvent.click(screen.getByTestId(selectors.wizard.nextButton));

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
