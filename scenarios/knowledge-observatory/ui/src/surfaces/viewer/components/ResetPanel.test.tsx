import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ResetPanel, type ResetPanelProps } from "./ResetPanel";

const createProps = (overrides: Partial<ResetPanelProps> = {}): ResetPanelProps => ({
  canReset: true,
  defaults: { maxAgeDays: 30, keepMinEntries: 3 },
  isBusy: false,
  resetError: "",
  result: null,
  onPreview: vi.fn(),
  onApply: vi.fn(),
  ...overrides,
});

describe("ResetPanel", () => {
  it("shows disabled message when reset is not supported", () => {
    render(<ResetPanel {...createProps({ canReset: false })} />);
    expect(screen.getByText(/Reset is available for PROBLEMS/i)).toBeDefined();
  });

  it("invokes preview handler with current values", () => {
    const onPreview = vi.fn();
    render(<ResetPanel {...createProps({ onPreview })} />);
    fireEvent.click(screen.getByRole("button", { name: /Preview/i }));
    expect(onPreview).toHaveBeenCalledWith({ maxAgeDays: 30, keepMinEntries: 3 });
  });
});
