import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useTabLikeNavigationShortcuts } from "../useTabLikeNavigationShortcuts";
import type { PaneMetadata } from "../../stores/useWorkspaceStore";

const panes: PaneMetadata[] = [
  {
    sessionId: "one",
    name: "One",
    headerColor: "transparent",
    themeId: "default",
    fontSize: 14,
    groupId: null,
    supportsMessagesView: false, manuallyUnread: false,
  },
  {
    sessionId: "two",
    name: "Two",
    headerColor: "transparent",
    themeId: "default",
    fontSize: 14,
    groupId: null,
    supportsMessagesView: false, manuallyUnread: false,
  },
];

function Harness({
  enabled,
  activePane = "one",
  onActivatePane,
  onClosePane,
}: {
  enabled: boolean;
  activePane?: string | null;
  onActivatePane: (sessionId: string) => void;
  onClosePane: (sessionId: string) => void;
}) {
  useTabLikeNavigationShortcuts({
    enabled,
    panes,
    activePane,
    onActivatePane,
    onClosePane,
  });
  return null;
}

describe("useTabLikeNavigationShortcuts", () => {
  it("cycles, jumps, and closes when enabled", () => {
    const onActivatePane = vi.fn();
    const onClosePane = vi.fn();
    render(<Harness enabled onActivatePane={onActivatePane} onClosePane={onClosePane} />);

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Tab", ctrlKey: true }));
    expect(onActivatePane).toHaveBeenLastCalledWith("two");

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "2", ctrlKey: true }));
    expect(onActivatePane).toHaveBeenLastCalledWith("two");

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "w", ctrlKey: true }));
    expect(onClosePane).toHaveBeenCalledWith("one");
  });

  it("does not handle shortcuts when disabled", () => {
    const onActivatePane = vi.fn();
    const onClosePane = vi.fn();
    render(<Harness enabled={false} onActivatePane={onActivatePane} onClosePane={onClosePane} />);

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Tab", ctrlKey: true }));
    window.dispatchEvent(new KeyboardEvent("keydown", { key: "w", ctrlKey: true }));

    expect(onActivatePane).not.toHaveBeenCalled();
    expect(onClosePane).not.toHaveBeenCalled();
  });
});
