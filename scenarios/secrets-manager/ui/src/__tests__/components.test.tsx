import { describe, it, expect, vi } from "vitest";
import "@testing-library/jest-dom/vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { TabNav } from "../components/ui/TabNav";
import { TabTip } from "../components/ui/TabTip";
import { ResourceTable } from "../sections/ResourceTable";
import { TutorialOverlay } from "../components/ui/TutorialOverlay";

describe("TabNav", () => {
  it("renders badges and invokes onChange", () => {
    const onChange = vi.fn();
    render(
      <TabNav
        tabs={[
          { id: "dashboard", label: "Dashboard", badgeCount: 2 },
          { id: "resources", label: "Resources" }
        ]}
        activeTab="dashboard"
        onChange={onChange}
      />
    );

    expect(screen.getByText("Dashboard")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Resources"));
    expect(onChange).toHaveBeenCalledWith("resources");
  });
});

describe("TabTip", () => {
  it("shows CTA and fires action", () => {
    const onAction = vi.fn();
    render(
      <TabTip
        title="3 tiers need strategies"
        description="Click through to fix them."
        actionLabel="Open"
        onAction={onAction}
      />
    );

    fireEvent.click(screen.getByText("Open"));
    expect(onAction).toHaveBeenCalled();
  });
});

describe("ResourceTable", () => {
  const sample = [
    {
      resource_name: "postgres",
      secrets_total: 3,
      secrets_found: 1,
      secrets_missing: 2,
      secrets_optional: 0,
      health_status: "critical",
      last_checked: new Date().toISOString()
    },
    {
      resource_name: "redis",
      secrets_total: 2,
      secrets_found: 2,
      secrets_missing: 0,
      secrets_optional: 0,
      health_status: "healthy",
      last_checked: new Date().toISOString()
    }
  ];

  it("filters by search and actionable toggle", async () => {
    const onOpenResource = vi.fn();
    render(<ResourceTable resourceStatuses={sample} isLoading={false} onOpenResource={onOpenResource} />);

    fireEvent.change(screen.getByPlaceholderText("Search resources"), { target: { value: "post" } });
    expect(screen.getByText("postgres")).toBeInTheDocument();
    expect(screen.queryByText("redis")).not.toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText("Search resources"), { target: { value: "" } });
    fireEvent.click(screen.getByText("Only action needed"));
    expect(screen.getByText("postgres")).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByText("redis")).not.toBeInTheDocument());
  });
});

describe("TutorialOverlay", () => {
  it("scrolls to anchor when provided", () => {
    const anchor = document.createElement("div");
    anchor.id = "anchor-test";
    anchor.scrollIntoView = vi.fn();
    document.body.appendChild(anchor);

    render(
      <TutorialOverlay
        title="Test"
        stepLabel="Step 1"
        content={<div>content</div>}
        anchorId="anchor-test"
        onClose={() => {}}
      />
    );

    expect(anchor.scrollIntoView).toHaveBeenCalled();
  });
});
