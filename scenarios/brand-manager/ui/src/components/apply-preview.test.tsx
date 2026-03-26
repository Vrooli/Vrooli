import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApplyPreview } from "./apply-preview";

// [REQ:BM-REQ-UI-APPLY]

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    fetchApplyPreview: vi.fn(),
  };
});

import { fetchApplyPreview } from "../lib/api";
const mockFetchApplyPreview = vi.mocked(fetchApplyPreview);

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

describe("ApplyPreview", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders section wrapper", () => {
    renderWithQuery(<ApplyPreview brandId="b1" />);
    expect(screen.getByTestId("apply-preview-section")).toBeTruthy();
  });

  it("renders scenario name input", () => {
    renderWithQuery(<ApplyPreview brandId="b1" />);
    expect(screen.getByTestId("apply-scenario-input")).toBeTruthy();
  });

  it("renders preview button disabled when input is empty", () => {
    renderWithQuery(<ApplyPreview brandId="b1" />);
    const btn = screen.getByTestId("apply-preview-btn");
    expect(btn.getAttribute("disabled")).not.toBeNull();
  });

  it("enables preview button when scenario name is entered", () => {
    renderWithQuery(<ApplyPreview brandId="b1" />);
    fireEvent.change(screen.getByTestId("apply-scenario-input"), {
      target: { value: "my-scenario" },
    });
    const btn = screen.getByTestId("apply-preview-btn");
    expect(btn.getAttribute("disabled")).toBeNull();
  });

  it("displays preview results after clicking preview", async () => {
    mockFetchApplyPreview.mockResolvedValue({
      scenario: "my-scenario",
      brand_id: "b1",
      brand_version: 1,
      applied: [
        { type: "css", file: "ui/src/index.css", element: "colors" },
        { type: "json", file: "ui/public/manifest.json", element: "name" },
      ],
      dry_run: true,
    });

    renderWithQuery(<ApplyPreview brandId="b1" />);
    fireEvent.change(screen.getByTestId("apply-scenario-input"), {
      target: { value: "my-scenario" },
    });
    fireEvent.click(screen.getByTestId("apply-preview-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("apply-preview-results")).toBeTruthy();
    });
    expect(screen.getByText("ui/src/index.css")).toBeTruthy();
    expect(screen.getByText("ui/public/manifest.json")).toBeTruthy();
  });

  it("displays skipped elements in preview", async () => {
    mockFetchApplyPreview.mockResolvedValue({
      scenario: "my-scenario",
      brand_id: "b1",
      brand_version: 1,
      applied: [],
      skipped: [{ element: "favicon", reason: "No source asset" }],
      dry_run: true,
    });

    renderWithQuery(<ApplyPreview brandId="b1" />);
    fireEvent.change(screen.getByTestId("apply-scenario-input"), {
      target: { value: "my-scenario" },
    });
    fireEvent.click(screen.getByTestId("apply-preview-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("apply-preview-results")).toBeTruthy();
    });
    expect(screen.getByText("favicon")).toBeTruthy();
    expect(screen.getByText(/No source asset/)).toBeTruthy();
  });

  it("shows error message on preview failure", async () => {
    mockFetchApplyPreview.mockRejectedValue(new Error("Scenario not found"));

    renderWithQuery(<ApplyPreview brandId="b1" />);
    fireEvent.change(screen.getByTestId("apply-scenario-input"), {
      target: { value: "bad-scenario" },
    });
    fireEvent.click(screen.getByTestId("apply-preview-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("apply-preview-error")).toBeTruthy();
    });
    expect(screen.getByText("Scenario not found")).toBeTruthy();
  });
});
