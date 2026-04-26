import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import StandardsPage from "./StandardsPage";

// [REQ:BM-REQ-API-STANDARDS] [REQ:BM-REQ-AUDIT-RULES] [REQ:BM-REQ-AUDIT-ENDPOINT]

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    fetchStandards: vi.fn(),
    evaluateRule: vi.fn(),
    evaluateAllRules: vi.fn(),
  };
});

import { fetchStandards, evaluateRule, evaluateAllRules } from "../lib/api";
const mockFetchStandards = vi.mocked(fetchStandards);
const mockEvaluateRule = vi.mocked(evaluateRule);
const mockEvaluateAllRules = vi.mocked(evaluateAllRules);

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

describe("StandardsPage", () => {
  const onNavigate = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders page with title", () => {
    mockFetchStandards.mockReturnValue(new Promise(() => {}));
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    expect(screen.getByTestId("standards-page")).toBeTruthy();
    expect(screen.getByText("Brand Standards")).toBeTruthy();
  });

  it("shows loading state", () => {
    mockFetchStandards.mockReturnValue(new Promise(() => {}));
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    expect(screen.getByTestId("standards-loading")).toBeTruthy();
  });

  it("renders standards list when loaded", async () => {
    mockFetchStandards.mockResolvedValue({
      rules: [
        { id: "has-logo", name: "Logo Required", description: "Every scenario must have a logo", severity: "error" },
        { id: "has-favicon", name: "Favicon Required", description: "Favicon must be set", severity: "warning" },
        { id: "has-colors", name: "Color System", description: "Color system must be defined", severity: "error" },
      ],
    });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("standards-list")).toBeTruthy();
    });

    expect(screen.getByText("Logo Required")).toBeTruthy();
    expect(screen.getByText("Favicon Required")).toBeTruthy();
    expect(screen.getByText("Color System")).toBeTruthy();
  });

  it("shows severity badges for each rule", async () => {
    mockFetchStandards.mockResolvedValue({
      rules: [
        { id: "r1", name: "Rule 1", description: "Desc", severity: "error" },
        { id: "r2", name: "Rule 2", description: "Desc", severity: "warning" },
      ],
    });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByText("error")).toBeTruthy();
      expect(screen.getByText("warning")).toBeTruthy();
    });
  });

  it("shows error state on API failure", async () => {
    mockFetchStandards.mockRejectedValue(new Error("API down"));
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("standards-error")).toBeTruthy();
    });
  });

  it("shows empty state when no rules", async () => {
    mockFetchStandards.mockResolvedValue({ rules: [] });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("standards-empty")).toBeTruthy();
    });
  });

  it("navigates back to brands", () => {
    mockFetchStandards.mockReturnValue(new Promise(() => {}));
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    fireEvent.click(screen.getByTestId("back-to-brands"));
    expect(onNavigate).toHaveBeenCalledWith("/brands");
  });

  it("renders rule descriptions", async () => {
    mockFetchStandards.mockResolvedValue({
      rules: [
        { id: "r1", name: "Logo", description: "Must have a logo file", severity: "error" },
      ],
    });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByText("Must have a logo file")).toBeTruthy();
    });
  });

  it("expands a rule card to show detailed metadata", async () => {
    mockFetchStandards.mockResolvedValue({
      rules: [
        {
          id: "has-logo",
          name: "Logo Required",
          description: "Must have a logo",
          severity: "warning",
          target_files: ["ui/public/logo.svg"],
          detailed_description: "Validates the logo path is non-empty.",
          fix_instructions: "1. Add a logo asset.",
        },
      ],
    });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => screen.getByTestId("standard-has-logo"));
    fireEvent.click(screen.getByTestId("standard-has-logo-toggle"));

    expect(screen.getByTestId("standard-has-logo-details")).toBeTruthy();
    expect(screen.getByText("Validates the logo path is non-empty.")).toBeTruthy();
    expect(screen.getByText("ui/public/logo.svg", { exact: false })).toBeTruthy();
  });

  it("supports multiple cards expanded simultaneously", async () => {
    mockFetchStandards.mockResolvedValue({
      rules: [
        { id: "r1", name: "R1", description: "d1", severity: "error", detailed_description: "DD1" },
        { id: "r2", name: "R2", description: "d2", severity: "warning", detailed_description: "DD2" },
      ],
    });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => screen.getByTestId("standard-r1"));
    fireEvent.click(screen.getByTestId("standard-r1-toggle"));
    fireEvent.click(screen.getByTestId("standard-r2-toggle"));

    expect(screen.getByTestId("standard-r1-details")).toBeTruthy();
    expect(screen.getByTestId("standard-r2-details")).toBeTruthy();
  });

  it("runs per-rule check using shared scenario input", async () => {
    mockFetchStandards.mockResolvedValue({
      rules: [{ id: "has-logo", name: "Logo", description: "d", severity: "warning" }],
    });
    mockEvaluateRule.mockResolvedValue({
      scenario: "demo",
      results: [{ rule_id: "has-logo", pass: true, severity: "warning", message: "Logo Present is defined" }],
    });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => screen.getByTestId("standard-has-logo"));
    fireEvent.change(screen.getByTestId("standards-scenario-input"), { target: { value: "demo" } });
    fireEvent.click(screen.getByTestId("standard-has-logo-toggle"));
    fireEvent.click(screen.getByTestId("standard-has-logo-check-btn"));

    await waitFor(() => screen.getByTestId("standard-has-logo-result"));
    expect(mockEvaluateRule).toHaveBeenCalledWith("demo", "has-logo");
    expect(screen.getByTestId("standard-has-logo-result").textContent).toContain("pass");
  });

  it("renders all-pass summary in green", async () => {
    mockFetchStandards.mockResolvedValue({
      rules: [{ id: "r1", name: "R1", description: "d", severity: "warning" }],
    });
    mockEvaluateAllRules.mockResolvedValue({
      scenario: "demo",
      results: [{ rule_id: "r1", pass: true, severity: "warning", message: "ok" }],
    });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => screen.getByTestId("standards-list"));
    fireEvent.change(screen.getByTestId("standards-scenario-input"), { target: { value: "demo" } });
    fireEvent.click(screen.getByTestId("standards-scan-all-btn"));

    await waitFor(() => screen.getByTestId("standards-scan-summary"));
    const summary = screen.getByTestId("standards-scan-summary");
    expect(summary.textContent).toContain("1 / 1");
    expect(summary.className).toContain("emerald");
  });

  it("colors summary red when an error-severity rule fails", async () => {
    mockFetchStandards.mockResolvedValue({
      rules: [
        { id: "e1", name: "E1", description: "d", severity: "error" },
        { id: "w1", name: "W1", description: "d", severity: "warning" },
      ],
    });
    mockEvaluateAllRules.mockResolvedValue({
      scenario: "demo",
      results: [
        { rule_id: "e1", pass: false, severity: "error", message: "missing" },
        { rule_id: "w1", pass: true, severity: "warning", message: "ok" },
      ],
    });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => screen.getByTestId("standards-list"));
    fireEvent.change(screen.getByTestId("standards-scenario-input"), { target: { value: "demo" } });
    fireEvent.click(screen.getByTestId("standards-scan-all-btn"));

    await waitFor(() => screen.getByTestId("standards-scan-summary"));
    expect(screen.getByTestId("standards-scan-summary").className).toContain("red");
  });

  it("colors summary amber when only non-error rules fail", async () => {
    mockFetchStandards.mockResolvedValue({
      rules: [
        { id: "w1", name: "W1", description: "d", severity: "warning" },
        { id: "i1", name: "I1", description: "d", severity: "info" },
      ],
    });
    mockEvaluateAllRules.mockResolvedValue({
      scenario: "demo",
      results: [
        { rule_id: "w1", pass: false, severity: "warning", message: "missing" },
        { rule_id: "i1", pass: true, severity: "info", message: "ok" },
      ],
    });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => screen.getByTestId("standards-list"));
    fireEvent.change(screen.getByTestId("standards-scenario-input"), { target: { value: "demo" } });
    fireEvent.click(screen.getByTestId("standards-scan-all-btn"));

    await waitFor(() => screen.getByTestId("standards-scan-summary"));
    expect(screen.getByTestId("standards-scan-summary").className).toContain("amber");
  });

  it("disables Check Scenario button when scenario input is empty", async () => {
    mockFetchStandards.mockResolvedValue({
      rules: [{ id: "r1", name: "R1", description: "d", severity: "warning" }],
    });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => screen.getByTestId("standard-r1"));
    fireEvent.click(screen.getByTestId("standard-r1-toggle"));

    const btn = screen.getByTestId("standard-r1-check-btn") as HTMLButtonElement;
    expect(btn.disabled).toBe(true);
  });

  it("renders rule IDs", async () => {
    mockFetchStandards.mockResolvedValue({
      rules: [
        { id: "has-logo", name: "Logo", description: "Desc", severity: "error" },
      ],
    });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByText("ID: has-logo")).toBeTruthy();
    });
  });
});
