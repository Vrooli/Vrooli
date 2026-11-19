import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { MessageList as MessageListV100 } from "@vrooli/react-component-library/MessageList/1.0.0";
import { MessageList as MessageListV111 } from "@vrooli/react-component-library/MessageList/1.1.1";

import { renderWithProviders } from "../test-utils";

/** A deliberately two-version page for the page-global stylesheet contract. */
function StyleSheetCollisionFixture() {
  return (
    <main data-testid="stylesheet-collision-fixture">
      <section data-testid="stylesheet-collision-v100">
        <MessageListV100 state="empty" height={180} />
      </section>
      <section data-testid="stylesheet-collision-v111">
        <MessageListV111 state="empty" height={180} />
      </section>
    </main>
  );
}

describe("StyleSheetCollisionFixture", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("mounts one stylesheet per exact MessageList version", () => {
    // Released MessageList composes a VirtualList inside AsyncBoundary, whose
    // authored child allowlist names only Card/EmptyState/ErrorState. The
    // warning is a real cross-asset contract conflict (see
    // docs/internal/PROBLEMS.md); this test owns the stylesheet invariant, so
    // it asserts the warning is surfaced rather than silently swallowed.
    const warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    renderWithProviders(<StyleSheetCollisionFixture />);

    expect(screen.getByTestId("stylesheet-collision-v100")).toBeInTheDocument();
    expect(screen.getByTestId("stylesheet-collision-v111")).toBeInTheDocument();
    const sheets = Array.from(
      document.head.querySelectorAll<HTMLStyleElement>("style[data-rcl-sheet]"),
    ).filter((sheet) => sheet.dataset.rclSheet?.includes("messagelist"));

    expect(sheets).toHaveLength(2);
    expect(sheets.map((sheet) => sheet.dataset.rclSheet)).toEqual([
      "react-component-library-messagelist-1.1.1",
      "react-component-library-messagelist-1.0.0",
    ]);
    expect(sheets[0]?.textContent).not.toBe(sheets[1]?.textContent);
    expect(warning).toHaveBeenCalledWith(
      expect.stringContaining("allowedChildComponents"),
    );
  });
});
