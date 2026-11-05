// [REQ:REQ-P0-003] App Accessibility Tests
import { screen, fireEvent } from "./test-utils";
import { renderWithQueryClient } from "./test-utils";
import App from "./App";

function renderApp() {
  return renderWithQueryClient(<App />);
}

/** Helper to get all application tabs and assert their stable order. */
function getTabs() {
  const tabs = screen.getAllByRole("tab");
  expect(tabs).toHaveLength(4);
  const [tab0, tab1, tab2, tab3] = tabs;
  if (!tab0 || !tab1 || !tab2 || !tab3) throw new Error("Expected 4 tabs");
  return [tab0, tab1, tab2, tab3] as const;
}

describe("App - Accessibility", () => {
  it("renders skip-to-content link", () => {
    renderApp();
    const skipLink = screen.getByTestId("skip-to-content");
    expect(skipLink).toBeInTheDocument();
    expect(skipLink).toHaveAttribute("href", "#main-content");
    expect(skipLink).toHaveTextContent("Skip to main content");
  });

  it("has main content landmark with id", () => {
    renderApp();
    const main = screen.getByRole("main");
    expect(main).toHaveAttribute("id", "app-shell-main");
  });

  it("renders navigation landmark", () => {
    renderApp();
    const nav = screen.getByRole("navigation", { name: /main navigation/i });
    expect(nav).toBeInTheDocument();
  });

  it("navigation uses tablist pattern", () => {
    renderApp();
    const tablist = screen.getByRole("tablist", { name: /application views/i });
    expect(tablist).toBeInTheDocument();
  });

  it("nav buttons use tab role with aria-selected", () => {
    renderApp();
    const [tab0, tab1, tab2] = getTabs();
    // First tab should be selected by default
    expect(tab0).toHaveAttribute("aria-selected", "true");
    expect(tab1).toHaveAttribute("aria-selected", "false");
    expect(tab2).toHaveAttribute("aria-selected", "false");
  });

  it("tabs have aria-controls linking to tabpanels", () => {
    renderApp();
    const [tab0, tab1, tab2] = getTabs();
    expect(tab0).toHaveAttribute("aria-controls", "tabpanel-wizard");
    expect(tab1).toHaveAttribute("aria-controls", "tabpanel-dashboard");
    expect(tab2).toHaveAttribute("aria-controls", "tabpanel-glossary");
    expect(getTabs()[3]).toHaveAttribute("aria-controls", "tabpanel-configuration");
  });

  it("all tabpanels exist in DOM with proper IDs", () => {
    renderApp();
    expect(document.getElementById("tabpanel-wizard")).toBeInTheDocument();
    expect(document.getElementById("tabpanel-dashboard")).toBeInTheDocument();
    expect(document.getElementById("tabpanel-glossary")).toBeInTheDocument();
    expect(document.getElementById("tabpanel-configuration")).toBeInTheDocument();
  });

  it("only active tabpanel is visible", () => {
    renderApp();
    const wizardPanel = document.getElementById("tabpanel-wizard");
    const dashboardPanel = document.getElementById("tabpanel-dashboard");
    const glossaryPanel = document.getElementById("tabpanel-glossary");
    const configurationPanel = document.getElementById("tabpanel-configuration");
    expect(wizardPanel).not.toHaveAttribute("hidden");
    expect(dashboardPanel).toHaveAttribute("hidden");
    expect(glossaryPanel).toHaveAttribute("hidden");
    expect(configurationPanel).toHaveAttribute("hidden");
  });

  it("switching tabs updates aria-selected and visible tabpanel", () => {
    renderApp();
    const [tab0, tab1] = getTabs();

    fireEvent.click(tab1);
    expect(tab0).toHaveAttribute("aria-selected", "false");
    expect(tab1).toHaveAttribute("aria-selected", "true");
    expect(document.getElementById("tabpanel-wizard")).toHaveAttribute("hidden");
    expect(document.getElementById("tabpanel-dashboard")).not.toHaveAttribute("hidden");
  });

  it("inactive tabs have tabIndex -1, active tab has tabIndex 0", () => {
    renderApp();
    const [tab0, tab1, tab2] = getTabs();
    expect(tab0).toHaveAttribute("tabindex", "0");
    expect(tab1).toHaveAttribute("tabindex", "-1");
    expect(tab2).toHaveAttribute("tabindex", "-1");
  });

  it("arrow keys navigate between tabs", () => {
    renderApp();
    const [tab0, tab1] = getTabs();
    tab0.focus();
    fireEvent.keyDown(tab0, { key: "ArrowRight" });
    expect(tab1).toHaveAttribute("aria-selected", "true");
    expect(tab0).toHaveAttribute("aria-selected", "false");
  });

  it("Home/End keys jump to first/last tab", () => {
    renderApp();
    const [tab0, , , tab3] = getTabs();
    tab0.focus();
    fireEvent.keyDown(tab0, { key: "End" });
    expect(tab3).toHaveAttribute("aria-selected", "true");
    fireEvent.keyDown(tab3, { key: "Home" });
    expect(tab0).toHaveAttribute("aria-selected", "true");
  });

  it("nav buttons have improved contrast (text-muted for inactive)", () => {
    renderApp();
    const [, tab1] = getTabs();
    expect(tab1.className).toContain("text-muted");
  });

  it("each view has an h1 heading for page-has-heading-one", () => {
    renderApp();
    // Wizard welcome has h1
    expect(screen.getByRole("heading", { level: 1, name: /welcome to vrooli/i })).toBeInTheDocument();

    // Switch to dashboard
    const [, dashboardTab, glossaryTab] = getTabs();
    fireEvent.click(dashboardTab);
    // Dashboard renders h1 (may be loading/empty state, but component renders h1)
    const dashH1 = screen.queryByRole("heading", { level: 1 });
    expect(dashH1).toBeInTheDocument();

    // Switch to glossary
    fireEvent.click(glossaryTab);
    expect(screen.getByRole("heading", { level: 1, name: /glossary/i })).toBeInTheDocument();
  });
});
