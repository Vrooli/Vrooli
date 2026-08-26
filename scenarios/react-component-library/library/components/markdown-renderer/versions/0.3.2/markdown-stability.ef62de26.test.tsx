/**
 * Render-stability tests for the markdown surface.
 *
 * These exist because of a user-visible defect: a message containing a mermaid
 * diagram flickered between its loading state and the rendered SVG roughly
 * every three seconds. The cause was not mermaid — it was that the whole
 * markdown subtree was being *unmounted and remounted* on every parent render,
 * because `MarkdownRenderer` rebuilt its `components` map whenever a caller
 * passed an inline callback (which every caller does). A new `code` function in
 * that map is a new React element *type*, and React responds to a changed type
 * by discarding the subtree and mounting a fresh one.
 *
 * So the invariants under test are about identity and mount counts, not about
 * pixels: the markdown subtree must survive a parent re-render, and a diagram
 * whose source has not changed must not be re-rendered from scratch.
 */
import { act, render, screen, waitFor } from "@testing-library/react";
import { useEffect, useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  MarkdownRenderer,
  resetMermaidRenderCacheForTests,
} from "./MarkdownRenderer.tsx";

// A counter incremented by mermaid's mocked render(). Because the module mock
// is hoisted, the counter has to live on globalThis rather than in a closure
// the factory can't see.
declare global {
  var __mermaidRenderCalls: number;
}

vi.mock("mermaid", () => ({
  default: {
    initialize: vi.fn(),
    render: vi.fn((id: string, code: string) => {
      globalThis.__mermaidRenderCalls = (globalThis.__mermaidRenderCalls || 0) + 1;
      return { svg: `<svg data-testid="rendered-svg" data-code="${code}"><title>${id}</title></svg>` };
    }),
  },
}));

const DIAGRAM = "graph TD;\n  A-->B;";
const MESSAGE = `Here is a diagram.\n\n\`\`\`mermaid\n${DIAGRAM}\n\`\`\`\n`;

/**
 * A parent that re-renders on demand and — critically — passes *inline*
 * callbacks to MarkdownRenderer, exactly as ChatMessageBubble does. If the
 * renderer is identity-stable this changes nothing downstream; if it is not,
 * the whole subtree remounts.
 */
function PollingParent({ content, ticks }: { content: string; ticks: number }) {
  const [tick, setTick] = useState(0);
  useEffect(() => {
    if (tick < ticks) setTick((value) => value + 1);
  }, [tick, ticks]);
  return (
    <div data-tick={tick}>
      <MarkdownRenderer
        content={content}
        resolveInlineToken={(text) => (text === "never" ? { href: "/never", kind: "entity" } : null)}
        onLinkClick={() => undefined}
        onMermaidOpen={() => undefined}
      />
    </div>
  );
}

describe("markdown render stability", () => {
  beforeEach(() => {
    globalThis.__mermaidRenderCalls = 0;
    resetMermaidRenderCacheForTests();
    vi.useFakeTimers({ shouldAdvanceTime: true });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders a mermaid diagram exactly once across many parent re-renders", async () => {
    render(<PollingParent content={MESSAGE} ticks={8} />);

    await act(() => vi.advanceTimersByTimeAsync(200));
    await waitFor(() => expect(screen.getByTestId("rendered-svg")).toBeInTheDocument());

    // Let every queued re-render settle, then confirm nothing re-rendered the
    // diagram. Before the fix this was 9 (one per parent render).
    await act(() => vi.advanceTimersByTimeAsync(500));
    expect(globalThis.__mermaidRenderCalls).toBe(1);
  });

  it("never returns to the loading state once a diagram has rendered", async () => {
    const { container } = render(<PollingParent content={MESSAGE} ticks={6} />);

    await act(() => vi.advanceTimersByTimeAsync(200));
    await waitFor(() => expect(screen.getByTestId("rendered-svg")).toBeInTheDocument());

    // The flicker the user reported: a re-render dropping the SVG and showing
    // the placeholder again. Poll across several parent renders and assert the
    // SVG is continuously present.
    for (let i = 0; i < 6; i++) {
      await act(() => vi.advanceTimersByTimeAsync(50));
      expect(container.querySelector("[data-testid='rendered-svg']")).not.toBeNull();
      expect(screen.queryByTestId("mermaid-skeleton")).toBeNull();
    }
  });

  it("keeps the same DOM node for the diagram across parent re-renders", async () => {
    const { container } = render(<PollingParent content={MESSAGE} ticks={5} />);

    await act(() => vi.advanceTimersByTimeAsync(200));
    await waitFor(() => expect(screen.getByTestId("rendered-svg")).toBeInTheDocument());
    const first = container.querySelector("[data-mermaid-host]");

    await act(() => vi.advanceTimersByTimeAsync(400));
    const last = container.querySelector("[data-mermaid-host]");

    // Node identity is the direct signal of "was this subtree remounted".
    expect(last).toBe(first);
  });

  it("re-renders when the diagram source actually changes", async () => {
    const { rerender } = render(<MarkdownRenderer content={MESSAGE} />);
    await act(() => vi.advanceTimersByTimeAsync(200));
    await waitFor(() => expect(globalThis.__mermaidRenderCalls).toBe(1));

    rerender(<MarkdownRenderer content={"```mermaid\ngraph LR;\n  X-->Y;\n```"} />);
    await act(() => vi.advanceTimersByTimeAsync(200));
    await waitFor(() => expect(globalThis.__mermaidRenderCalls).toBe(2));
  });

  it("renders identical diagrams in two messages with a single mermaid pass", async () => {
    render(
      <>
        <MarkdownRenderer content={MESSAGE} />
        <MarkdownRenderer content={MESSAGE} />
      </>,
    );

    await act(() => vi.advanceTimersByTimeAsync(200));
    await waitFor(() => expect(screen.getAllByTestId("rendered-svg")).toHaveLength(2));

    // Two mounts of the same source must coalesce onto one render pass.
    expect(globalThis.__mermaidRenderCalls).toBe(1);
  });
});
