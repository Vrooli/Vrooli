// [REQ:REQ-P0-003] Wizard UI Flow
import { screen, fireEvent } from "@testing-library/react";
import { renderWithQueryClient } from "./test-utils";
import App from "./App";

function renderApp() {
  return renderWithQueryClient(<App />);
}

describe("App - Wizard Navigation", () => {
  it("renders the wizard shell", () => {
    renderApp();
    expect(screen.getByTestId("wizard-shell")).toBeInTheDocument();
  });

  it("starts on the welcome step", () => {
    renderApp();
    expect(screen.getByText(/welcome to vrooli/i)).toBeInTheDocument();
  });

  it("shows Get Started button on welcome step", () => {
    renderApp();
    const nextButton = screen.getByTestId("wizard-next");
    expect(nextButton).toHaveTextContent(/get started/i);
  });

  it("advances to resource selection on Get Started click", () => {
    renderApp();
    fireEvent.click(screen.getByTestId("wizard-next"));
    // After advancing, the step-resources-loading should appear (waiting for API data)
    expect(screen.getByTestId("step-resources-loading")).toBeInTheDocument();
  });

  it("disables Next when no resources are selected", () => {
    renderApp();
    fireEvent.click(screen.getByTestId("wizard-next")); // go to step 2
    const nextButton = screen.getByTestId("wizard-next");
    expect(nextButton).toBeDisabled();
  });

  it("shows Back button on step 2", () => {
    renderApp();
    fireEvent.click(screen.getByTestId("wizard-next")); // go to step 2
    expect(screen.getByTestId("wizard-prev")).toBeInTheDocument();
  });

  it("navigates back to welcome from step 2", () => {
    renderApp();
    fireEvent.click(screen.getByTestId("wizard-next")); // go to step 2
    fireEvent.click(screen.getByTestId("wizard-prev")); // go back
    expect(screen.getByText(/welcome to vrooli/i)).toBeInTheDocument();
  });

  it("renders step announcement for screen readers", () => {
    renderApp();
    const announcement = screen.getByTestId("step-announcement");
    expect(announcement).toHaveTextContent("Step 1 of 4");
    expect(announcement).toHaveAttribute("aria-live", "assertive");
  });

  it("updates step announcement on navigation", () => {
    renderApp();
    fireEvent.click(screen.getByTestId("wizard-next")); // go to step 2
    expect(screen.getByTestId("step-announcement")).toHaveTextContent("Step 2 of 4");
  });
});

describe("App - View Navigation", () => {
  it("renders navigation bar", () => {
    renderApp();
    expect(screen.getByTestId("app-nav")).toBeInTheDocument();
  });

  it("shows wizard view by default", () => {
    renderApp();
    expect(screen.getByTestId("wizard-shell")).toBeInTheDocument();
  });

  it("shows keyboard shortcut hints on nav tabs", () => {
    renderApp();
    const nav = screen.getByTestId("app-nav");
    // kbd elements are aria-hidden, so query the DOM directly
    const kbds = nav.querySelectorAll("kbd");
    expect(kbds.length).toBe(3);
    expect(kbds[0]?.textContent).toContain("Alt+1");
    expect(kbds[1]?.textContent).toContain("Alt+2");
    expect(kbds[2]?.textContent).toContain("Alt+3");
  });

  it("switches view with Alt+number keyboard shortcut", () => {
    renderApp();
    // Start on wizard view
    expect(screen.getByTestId("wizard-shell")).toBeInTheDocument();
    // Press Alt+2 to switch to dashboard
    fireEvent.keyDown(document, { key: "2", altKey: true });
    expect(screen.getByTestId("health-dashboard")).toBeInTheDocument();
  });
});

describe("App - Step Transitions", () => {
  it("applies step entrance animation class on step content", () => {
    renderApp();
    const stepContent = screen.getByTestId("step-welcome").parentElement;
    expect(stepContent?.classList.contains("animate-step-enter")).toBe(true);
  });
});
