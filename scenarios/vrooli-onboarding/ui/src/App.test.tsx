// [REQ:REQ-P0-003] Wizard UI Flow
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithQueryClient } from "./test-utils";
import App from "./App";

beforeEach(() => {
  window.history.replaceState({}, "", "/");
});

async function renderApp() {
  const steps = [
    "welcome",
    "scenarios",
    "core-set",
    "resources",
    "credentials",
    "integrations",
    "host",
    "operating-mode",
    "apply",
    "validation",
  ].map((id, ordinal) => ({
    id,
    ordinal,
    title: id,
    route: `/setup/${id}`,
    deferred: false,
  }));
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () =>
      Promise.resolve({
        steps,
        version: "1.0.0",
        updated_at: "now",
        scenarios: {},
      }),
  });
  const result = renderWithQueryClient(<App />);
  if (window.location.pathname === "/health-dashboard") {
    await waitFor(() => expect(globalThis.fetch).toHaveBeenCalled());
  } else {
    await waitFor(() =>
      expect(screen.getByTestId("step-indicator-0")).toBeInTheDocument(),
    );
  }
  return result;
}

describe("App - Wizard Navigation", () => {
  it("renders the wizard shell", async () => {
    await renderApp();
    expect(screen.getByTestId("wizard-shell")).toBeInTheDocument();
  });

  it("starts on the welcome step", async () => {
    await renderApp();
    expect(screen.getByText(/welcome to vrooli/i)).toBeInTheDocument();
  });

  it("shows Get Started button on welcome step", async () => {
    await renderApp();
    const nextButton = screen.getByTestId("wizard-next");
    expect(nextButton).toHaveTextContent(/get started/i);
  });

  it("advances to resource selection on Get Started click", async () => {
    await renderApp();
    fireEvent.click(screen.getByTestId("wizard-next"));
    // V2 advances to manifest-derived scenario selection.
    expect(screen.getByTestId("step-select-scenarios")).toBeInTheDocument();
  });

  it("disables Next when no resources are selected", async () => {
    await renderApp();
    fireEvent.click(screen.getByTestId("wizard-next")); // go to step 2
    const nextButton = screen.getByTestId("wizard-next");
    expect(nextButton).toBeDisabled();
  });

  it("shows Back button on step 2", async () => {
    await renderApp();
    fireEvent.click(screen.getByTestId("wizard-next")); // go to step 2
    expect(screen.getByTestId("wizard-prev")).toBeInTheDocument();
  });

  it("navigates back to welcome from step 2", async () => {
    await renderApp();
    fireEvent.click(screen.getByTestId("wizard-next")); // go to step 2
    fireEvent.click(screen.getByTestId("wizard-prev")); // go back
    expect(screen.getByText(/welcome to vrooli/i)).toBeInTheDocument();
  });

  it("renders step announcement for screen readers", async () => {
    await renderApp();
    const announcement = screen.getByTestId("step-announcement");
    expect(announcement).toHaveTextContent("Step 1 of 10");
    expect(announcement).toHaveAttribute("aria-live", "assertive");
  });

  it("updates step announcement on navigation", async () => {
    await renderApp();
    fireEvent.click(screen.getByTestId("wizard-next")); // go to step 2
    expect(screen.getByTestId("step-announcement")).toHaveTextContent(
      "Step 2 of 10",
    );
  });
});

describe("App - View Navigation", () => {
  it("renders navigation bar", async () => {
    await renderApp();
    expect(screen.getByTestId("app-nav")).toBeInTheDocument();
  });

  it("shows wizard view by default", async () => {
    await renderApp();
    expect(screen.getByTestId("wizard-shell")).toBeInTheDocument();
  });

  it("opens the health dashboard on its shareable route", async () => {
    window.history.pushState({}, "", "/health-dashboard");
    await renderApp();
    expect(screen.getByTestId("health-dashboard")).toBeInTheDocument();
    window.history.pushState({}, "", "/");
  });

  it("shows keyboard shortcut hints on nav tabs", async () => {
    await renderApp();
    const nav = screen.getByTestId("app-nav");
    // kbd elements are aria-hidden, so query the DOM directly
    const kbds = nav.querySelectorAll("kbd");
    expect(kbds.length).toBe(3);
    expect(kbds[0]?.textContent).toContain("Alt+1");
    expect(kbds[1]?.textContent).toContain("Alt+2");
    expect(kbds[2]?.textContent).toContain("Alt+3");
  });

  it("switches view with Alt+number keyboard shortcut", async () => {
    await renderApp();
    // Start on wizard view
    expect(screen.getByTestId("wizard-shell")).toBeInTheDocument();
    // Press Alt+2 to switch to dashboard
    fireEvent.keyDown(document, { key: "2", altKey: true });
    expect(screen.getByTestId("health-dashboard")).toBeInTheDocument();
  });

  it("ignores Alt+number shortcut when input is focused", async () => {
    await renderApp();
    expect(screen.getByTestId("wizard-shell")).toBeInTheDocument();

    // Create and focus an input element
    const input = document.createElement("input");
    document.body.appendChild(input);
    input.focus();

    // Alt+2 with input focused should NOT switch views
    fireEvent.keyDown(input, { key: "2", altKey: true });
    expect(screen.getByTestId("wizard-shell")).toBeInTheDocument();

    document.body.removeChild(input);
  });

  it("ignores shortcut with Ctrl or Meta modifier", async () => {
    await renderApp();
    expect(screen.getByTestId("wizard-shell")).toBeInTheDocument();

    // Alt+Ctrl+2 should be ignored
    fireEvent.keyDown(document, { key: "2", altKey: true, ctrlKey: true });
    expect(screen.getByTestId("wizard-shell")).toBeInTheDocument();
  });
});

describe("App - Step Transitions", () => {
  it("applies step entrance animation class on step content", async () => {
    await renderApp();
    const stepContent = screen.getByTestId("step-welcome").parentElement;
    expect(stepContent?.classList.contains("animate-step-enter")).toBe(true);
  });
});

describe("App - View Switching", () => {
  it("switches to dashboard and back to wizard", async () => {
    await renderApp();
    fireEvent.click(screen.getByTestId("nav-dashboard"));
    expect(screen.getByTestId("health-dashboard")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("nav-wizard"));
    expect(screen.getByTestId("wizard-shell")).toBeInTheDocument();
  });

  it("switches to glossary view", async () => {
    await renderApp();
    fireEvent.click(screen.getByTestId("nav-glossary"));
    expect(screen.getByTestId("glossary-panel")).toBeInTheDocument();
  });

  it("step announcement is empty when not on wizard view", async () => {
    await renderApp();
    fireEvent.click(screen.getByTestId("nav-dashboard"));
    expect(screen.getByTestId("step-announcement")).toHaveTextContent("");
  });
});
