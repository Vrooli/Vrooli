import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import ScannerPage from "./ScannerPage";

// [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON] [REQ:BM-REQ-AUDIT-ENDPOINT]
// [REQ:BM-REQ-DISC-SCAN] [REQ:BM-REQ-SCAN-PARTIAL] [REQ:BM-REQ-AUDIT-RULES]

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    scanScenario: vi.fn(),
    fetchAuditRules: vi.fn(),
    evaluateScenario: vi.fn(),
  };
});

import { scanScenario, fetchAuditRules, evaluateScenario } from "../lib/api";
const mockScan = vi.mocked(scanScenario);
const mockRules = vi.mocked(fetchAuditRules);
const mockEvaluate = vi.mocked(evaluateScenario);

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

describe("ScannerPage", () => {
  const onNavigate = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mockRules.mockResolvedValue({
      rules: [
        { id: "has-logo", name: "Has Logo", description: "Scenario must have a logo", severity: "error" },
        { id: "has-colors", name: "Has Colors", description: "Brand colors defined", severity: "warning" },
      ],
    });
    mockScan.mockResolvedValue({
      scenario: "test-app",
      findings: [
        { file: "brand.css", element: "colors", type: "css", line: 10 },
        { file: "manifest.json", element: "identity", type: "json" },
      ],
      summary: { total: 2, css: 1, json: 1 },
    });
    mockEvaluate.mockResolvedValue({
      scenario: "test-app",
      results: [
        { rule_id: "has-logo", passed: true, message: "Logo found" },
        { rule_id: "has-colors", passed: false, message: "No color system" },
      ],
      pass_all: false,
    });
  });

  it("renders the scanner page with input", () => {
    renderWithQuery(<ScannerPage onNavigate={onNavigate} />);
    expect(screen.getByTestId("scanner-page")).toBeTruthy();
    expect(screen.getByTestId("scanner-input")).toBeTruthy();
    expect(screen.getByTestId("scan-btn")).toBeTruthy();
  });

  it("scan button is disabled when input is empty", () => {
    renderWithQuery(<ScannerPage onNavigate={onNavigate} />);
    const btn = screen.getByTestId("scan-btn");
    expect(btn.getAttribute("disabled")).not.toBeNull();
  });

  it("triggers scan on button click and shows results", async () => {
    renderWithQuery(<ScannerPage onNavigate={onNavigate} />);

    fireEvent.change(screen.getByTestId("scanner-input"), { target: { value: "test-app" } });
    fireEvent.click(screen.getByTestId("scan-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("scan-results")).toBeTruthy();
    });

    expect(screen.getByTestId("scan-total").textContent).toBe("2");
    expect(screen.getByTestId("scan-css").textContent).toBe("1");
    expect(screen.getByTestId("scan-json").textContent).toBe("1");
  });

  it("shows scan findings list", async () => {
    renderWithQuery(<ScannerPage onNavigate={onNavigate} />);

    fireEvent.change(screen.getByTestId("scanner-input"), { target: { value: "test-app" } });
    fireEvent.click(screen.getByTestId("scan-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("scan-findings")).toBeTruthy();
    });

    expect(screen.getByText("colors")).toBeTruthy();
    expect(screen.getByText(/brand\.css/)).toBeTruthy();
  });

  it("shows audit results with pass/fail", async () => {
    renderWithQuery(<ScannerPage onNavigate={onNavigate} />);

    fireEvent.change(screen.getByTestId("scanner-input"), { target: { value: "test-app" } });
    fireEvent.click(screen.getByTestId("scan-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("audit-results")).toBeTruthy();
    });

    expect(screen.getByTestId("audit-fail")).toBeTruthy();
    expect(screen.getByTestId("audit-items")).toBeTruthy();
  });

  it("shows audit rules section", async () => {
    renderWithQuery(<ScannerPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("audit-rules-section")).toBeTruthy();
    });

    expect(screen.getByText("Has Logo")).toBeTruthy();
    expect(screen.getByText("Has Colors")).toBeTruthy();
  });

  it("shows scan error on failure", async () => {
    mockScan.mockRejectedValue(new Error("Not found"));
    renderWithQuery(<ScannerPage onNavigate={onNavigate} />);

    fireEvent.change(screen.getByTestId("scanner-input"), { target: { value: "bad-scenario" } });
    fireEvent.click(screen.getByTestId("scan-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("scan-error")).toBeTruthy();
    });
  });

  it("navigates back to brands", () => {
    renderWithQuery(<ScannerPage onNavigate={onNavigate} />);

    fireEvent.click(screen.getByTestId("back-to-brands"));
    expect(onNavigate).toHaveBeenCalledWith("/brands");
  });

  it("shows empty findings message when no markers found", async () => {
    mockScan.mockResolvedValue({
      scenario: "clean-app",
      findings: [],
      summary: { total: 0, css: 0, json: 0 },
    });
    renderWithQuery(<ScannerPage onNavigate={onNavigate} />);

    fireEvent.change(screen.getByTestId("scanner-input"), { target: { value: "clean-app" } });
    fireEvent.click(screen.getByTestId("scan-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("scan-no-findings")).toBeTruthy();
    });
  });

  it("shows passing audit when all checks pass", async () => {
    mockEvaluate.mockResolvedValue({
      scenario: "good-app",
      results: [{ rule_id: "has-logo", passed: true, message: "OK" }],
      pass_all: true,
    });
    renderWithQuery(<ScannerPage onNavigate={onNavigate} />);

    fireEvent.change(screen.getByTestId("scanner-input"), { target: { value: "good-app" } });
    fireEvent.click(screen.getByTestId("scan-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("audit-pass")).toBeTruthy();
    });
  });

  it("triggers scan on Enter key", async () => {
    renderWithQuery(<ScannerPage onNavigate={onNavigate} />);

    const input = screen.getByTestId("scanner-input");
    fireEvent.change(input, { target: { value: "test-app" } });
    fireEvent.keyDown(input, { key: "Enter" });

    await waitFor(() => {
      expect(mockScan).toHaveBeenCalledWith("test-app");
    });
  });
});
