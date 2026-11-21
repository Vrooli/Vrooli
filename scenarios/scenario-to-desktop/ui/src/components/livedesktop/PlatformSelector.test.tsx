import { fireEvent, render, screen } from "@/test-utils";
import { describe, expect, it, vi } from "vitest";
import { Platform } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import { PlatformSelector } from "./PlatformSelector";

describe("live desktop PlatformSelector", () => {
  it("selects Linux and communicates unavailable target platforms", () => {
    const onChange = vi.fn();
    render(<PlatformSelector value={Platform.LINUX} onChange={onChange} />);

    const linux = screen.getByRole("button", { name: "Linux" });
    expect(linux).not.toBeDisabled();
    expect(screen.getByRole("button", { name: /Windows/ })).toBeDisabled();
    expect(screen.getByRole("button", { name: /macOS/ })).toBeDisabled();
    expect(screen.getAllByText("Soon")).toHaveLength(2);
    fireEvent.click(linux);
    expect(onChange).toHaveBeenCalledWith(Platform.LINUX);
  });

  it("does not falsely mark Linux as selected for another platform", () => {
    render(<PlatformSelector value={Platform.WIN} onChange={vi.fn()} />);
    expect(
      screen.getByRole("button", { name: "Linux" }).className,
    ).not.toContain("border-blue-500");
  });
});
