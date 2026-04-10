import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ExecutionPromptTab } from "./execution-prompt-tab";
import { selectors } from "../../consts/selectors";
import type { PromptTrace } from "../../types";

const makeTrace = (overrides?: Partial<PromptTrace>): PromptTrace => ({
  purpose: "workshop",
  prompt: "Do the workshop thing",
  used_fallback: false,
  captured_at: "2026-03-20T00:00:00Z",
  ...overrides,
});

describe("ExecutionPromptTab", () => {
  it("shows loading state", () => {
    render(<ExecutionPromptTab trace={null} isLoading={true} />);
    expect(screen.getByText(/Loading prompt trace/)).toBeInTheDocument();
  });

  it("shows empty state when no trace", () => {
    render(<ExecutionPromptTab trace={null} isLoading={false} />);
    expect(screen.getByTestId(selectors.executionDetails.promptEmpty)).toBeInTheDocument();
    expect(screen.getByText(/No prompt trace available/)).toBeInTheDocument();
  });

  it("renders trace with purpose and prompt", () => {
    render(<ExecutionPromptTab trace={makeTrace()} isLoading={false} />);
    expect(screen.getByTestId(selectors.executionDetails.promptTrace)).toBeInTheDocument();
    expect(screen.getByText("workshop")).toBeInTheDocument();
    expect(screen.getByText("Do the workshop thing")).toBeInTheDocument();
  });

  it("renders prompt revision when present", () => {
    render(
      <ExecutionPromptTab
        trace={makeTrace({ prompt_revision: "Updated prompt" })}
        isLoading={false}
      />,
    );
    expect(screen.getByText("Updated prompt")).toBeInTheDocument();
  });

  it("shows fallback indicator when used_fallback is true", () => {
    render(
      <ExecutionPromptTab
        trace={makeTrace({ used_fallback: true })}
        isLoading={false}
      />,
    );
    expect(screen.getByText("Fallback used")).toBeInTheDocument();
  });

  it("hides fallback indicator when not used", () => {
    render(
      <ExecutionPromptTab
        trace={makeTrace({ used_fallback: false })}
        isLoading={false}
      />,
    );
    expect(screen.queryByText("Fallback used")).not.toBeInTheDocument();
  });

  it("shows captured timestamp", () => {
    render(
      <ExecutionPromptTab
        trace={makeTrace({ captured_at: "2026-03-20T12:00:00Z" })}
        isLoading={false}
      />,
    );
    expect(screen.getByText(/Captured:.*2026-03-20T12:00:00Z/)).toBeInTheDocument();
  });
});
