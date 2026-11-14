import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@/test-utils";
import { CapturesSection } from "./CapturesSection";

const mockFetchSummary = vi.fn();
const mockOpen = vi.fn();
let mockSummary: { count: number; total_bytes: number } | null = null;

vi.mock("../../store/capturesStore", () => ({
  useCapturesStore: Object.assign(
    (selector: (s: Record<string, unknown>) => unknown) =>
      selector({
        summary: mockSummary,
        fetchSummary: mockFetchSummary,
        open: mockOpen,
      }),
    {
      getState: () => ({
        open: mockOpen,
      }),
    },
  ),
}));

beforeEach(() => {
  mockSummary = null;
  mockFetchSummary.mockClear();
  mockOpen.mockClear();
});

describe("CapturesSection", () => {
  it("shows empty state when count is 0", () => {
    mockSummary = { count: 0, total_bytes: 0 };
    render(<CapturesSection scenarioName="my-app" />);
    expect(screen.getByText("No captures yet")).toBeInTheDocument();
    expect(screen.queryByText("View All")).not.toBeInTheDocument();
  });

  it("renders count and size from summary", () => {
    mockSummary = { count: 3, total_bytes: 12_400_000 };
    render(<CapturesSection scenarioName="my-app" />);
    expect(screen.getByText("3 captures")).toBeInTheDocument();
    expect(screen.getByText("View All")).toBeInTheDocument();
  });

  it("calls fetchSummary on mount", () => {
    render(<CapturesSection scenarioName="my-app" />);
    expect(mockFetchSummary).toHaveBeenCalledWith("my-app");
  });

  it("shows singular form for 1 capture", () => {
    mockSummary = { count: 1, total_bytes: 1024 };
    render(<CapturesSection scenarioName="my-app" />);
    expect(screen.getByText("1 capture")).toBeInTheDocument();
  });
});
