import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import "@testing-library/jest-dom/vitest";

const mockCreateInformation = vi.fn();

vi.mock("../lib/api", () => ({
  createInformation: (...args: unknown[]) => mockCreateInformation(...args) as unknown,
  ApiRequestError: class ApiRequestError extends Error {
    status: number;
    category: string;
    retryable: boolean;
    constructor(status: number, apiError: { category: string; message: string; retryable: boolean }) {
      super(apiError.message);
      this.name = "ApiRequestError";
      this.status = status;
      this.category = apiError.category;
      this.retryable = apiError.retryable;
    }
  },
}));

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:3000/api/v1",
  buildApiUrl: (path: string) => `http://localhost:3000/api/v1${path}`,
}));

import { TextCapture } from "./TextCapture";

function renderComponent(schemeId = "scheme-1") {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <TextCapture schemeId={schemeId} />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  mockCreateInformation.mockReset();
});

// [REQ:P0-001] [REQ:P0-002] Zero-friction text capture with auto-focus
describe("TextCapture", () => {
  it("renders the capture container and input", () => {
    renderComponent();
    expect(screen.getByTestId("text-capture")).toBeInTheDocument();
    expect(screen.getByTestId("text-capture-input")).toBeInTheDocument();
  });

  it("renders the send button", () => {
    renderComponent();
    expect(screen.getByTestId("text-capture-send")).toBeInTheDocument();
  });

  it("disables send when input is empty", () => {
    renderComponent();
    expect(screen.getByTestId("text-capture-send")).toBeDisabled();
  });

  it("enables send when text is entered", () => {
    renderComponent();
    fireEvent.change(screen.getByTestId("text-capture-input"), {
      target: { value: "hello" },
    });
    expect(screen.getByTestId("text-capture-send")).not.toBeDisabled();
  });

  it("submits on Enter key", async () => {
    mockCreateInformation.mockResolvedValue({
      id: "i1",
      scheme_id: "scheme-1",
      type: "text",
      content: "test thought",
      canvas_x: 0,
      canvas_y: 0,
      created_at: "",
      updated_at: "",
    });
    renderComponent();
    const input = screen.getByTestId("text-capture-input");
    fireEvent.change(input, { target: { value: "test thought" } });
    fireEvent.keyDown(input, { key: "Enter" });
    await waitFor(() => {
      expect(mockCreateInformation).toHaveBeenCalled();
    });
  });

  it("does not submit on Shift+Enter", () => {
    renderComponent();
    const input = screen.getByTestId("text-capture-input");
    fireEvent.change(input, { target: { value: "multi-line" } });
    fireEvent.keyDown(input, { key: "Enter", shiftKey: true });
    expect(mockCreateInformation).not.toHaveBeenCalled();
  });

  it("clears input after successful submission", async () => {
    mockCreateInformation.mockResolvedValue({
      id: "i1",
      scheme_id: "scheme-1",
      type: "text",
      content: "cleared",
      canvas_x: 0,
      canvas_y: 0,
      created_at: "",
      updated_at: "",
    });
    renderComponent();
    const input = screen.getByTestId<HTMLTextAreaElement>("text-capture-input");
    fireEvent.change(input, { target: { value: "cleared" } });
    fireEvent.click(screen.getByTestId("text-capture-send"));
    await waitFor(() => {
      expect(input.value).toBe("");
    });
  });

  it("shows error banner on submission failure", async () => {
    mockCreateInformation.mockRejectedValue(new Error("Server error"));
    renderComponent();
    const input = screen.getByTestId("text-capture-input");
    fireEvent.change(input, { target: { value: "fail" } });
    fireEvent.click(screen.getByTestId("text-capture-send"));
    await waitFor(
      () => {
        expect(screen.getByTestId("error-banner")).toBeInTheDocument();
      },
      { timeout: 3000 },
    );
  });
});
