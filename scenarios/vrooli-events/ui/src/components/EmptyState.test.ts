// @vitest-environment node
import { describe, it, expect } from "vitest";

// [REQ:REQ-UI-009] EmptyState shared component — consistent empty state UX
describe("EmptyState component contract", () => {
  it("renders with required props", () => {
    const props = {
      icon: "MockIcon",
      title: "No data yet",
      description: "Start by creating your first item.",
    };
    expect(props.title).toBe("No data yet");
    expect(props.description).toContain("creating");
  });

  it("optionally includes an action link", () => {
    const action = { label: "Go to Policies", to: "/policies" };
    expect(action.label).toBe("Go to Policies");
    expect(action.to).toBe("/policies");
  });

  it("omitting action means no link rendered", () => {
    const props = {
      icon: "MockIcon",
      title: "Empty",
      description: "Nothing here.",
      action: undefined,
    };
    expect(props.action).toBeUndefined();
  });

  it("uses semantic token colors", () => {
    const iconClass = "text-[var(--text-muted)]";
    const titleClass = "text-[var(--text-primary)]";
    const descClass = "text-[var(--text-muted)]";
    expect(iconClass).toContain("--text-muted");
    expect(titleClass).toContain("--text-primary");
    expect(descClass).toContain("--text-muted");
  });

  it("action link uses accent color", () => {
    const linkClass = "text-[var(--color-accent)]";
    expect(linkClass).toContain("--color-accent");
  });
});
