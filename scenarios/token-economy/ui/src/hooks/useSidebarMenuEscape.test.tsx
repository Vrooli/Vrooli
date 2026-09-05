import { describe, expect, it, vi } from "vitest";
import { fireEvent } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { useSidebarMenuEscape } from "./useSidebarMenuEscape";

function Harness({ enabled, onEscape }: { enabled: boolean; onEscape: () => void }) {
  useSidebarMenuEscape(enabled, onEscape);
  return <div>ready</div>;
}

describe("useSidebarMenuEscape", () => {
  it("runs the callback only when enabled and Escape is pressed", () => {
    const onEscape = vi.fn();
    renderWithProviders(<Harness enabled={true} onEscape={onEscape} />);

    fireEvent.keyDown(window, { key: "Enter" });
    fireEvent.keyDown(window, { key: "Escape" });

    expect(onEscape).toHaveBeenCalledTimes(1);
  });

  it("does not listen while disabled", () => {
    const onEscape = vi.fn();
    renderWithProviders(<Harness enabled={false} onEscape={onEscape} />);

    fireEvent.keyDown(window, { key: "Escape" });

    expect(onEscape).not.toHaveBeenCalled();
  });
});
