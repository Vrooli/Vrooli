import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { TargetSizeField } from "./target-size-field";

const baseProps = {
  label: "Target size",
  kbLabel: "KB",
  mbLabel: "MB",
  noLimitLabel: "No limit",
} as const;

describe("TargetSizeField", () => {
  it("shows the no-limit helper at zero bytes", () => {
    render(<TargetSizeField {...baseProps} valueBytes={0} onChange={vi.fn()} data-testid="ts" />);
    // Regex matcher (allowed) — the helper renders the passed noLimitLabel prop.
    expect(screen.getByText(/no limit/i)).toBeInTheDocument();
  });

  it("emits bytes = amount × 1024 in KB mode", () => {
    const onChange = vi.fn();
    render(<TargetSizeField {...baseProps} valueBytes={0} onChange={onChange} data-testid="ts" />);
    fireEvent.change(screen.getByTestId("ts"), { target: { value: "200" } });
    expect(onChange).toHaveBeenCalledWith(200 * 1024);
  });

  it("re-emits in MB when the unit switches", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <TargetSizeField {...baseProps} valueBytes={500 * 1024} onChange={onChange} data-testid="ts" />,
    );
    await user.click(screen.getByRole("radio", { name: "MB" }));
    // 500 KB / 1024 ≈ 0.49 MB → 0.49 * 1MB bytes
    expect(onChange).toHaveBeenCalled();
    const lastCall = onChange.mock.calls.at(-1);
    expect(lastCall?.[0]).toBeGreaterThan(0);
  });

  it("treats a non-positive amount as no limit (0 bytes)", () => {
    const onChange = vi.fn();
    render(
      <TargetSizeField {...baseProps} valueBytes={100 * 1024} onChange={onChange} data-testid="ts" />,
    );
    fireEvent.change(screen.getByTestId("ts"), { target: { value: "0" } });
    expect(onChange).toHaveBeenCalledWith(0);
  });
});
