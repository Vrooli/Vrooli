import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import {
  IntegrationsOverview,
  type IntegrationGroup,
  type IntegrationItem,
} from "@vrooli/react-component-library/IntegrationsOverview/1.0.5";

function item(overrides: Partial<IntegrationItem> & Pick<IntegrationItem, "id" | "name">): IntegrationItem {
  return { status: "ready", ...overrides };
}

function groups(...items: IntegrationItem[]): IntegrationGroup[] {
  return [{ id: "apps", label: "Powered by other apps", items }];
}

const summary = () => document.querySelector("[data-rcl-integrations-summary-title]")?.textContent;

describe("IntegrationsOverview", () => {
  it("says everything is ready when it is", () => {
    render(<IntegrationsOverview groups={groups(item({ id: "a", name: "Voice" }), item({ id: "b", name: "Notifications" }))} />);
    expect(summary()).toBe("All 2 ready");
  });

  it("counts only fully working items as ready", () => {
    render(
      <IntegrationsOverview
        groups={groups(
          item({ id: "a", name: "Voice", status: "limited" }),
          item({ id: "b", name: "Grok", status: "missing" }),
          item({ id: "c", name: "Codex" }),
        )}
      />,
    );
    expect(summary()).toBe("1 of 3 ready");
  });

  it("lists a broken item under the summary with its one fix", () => {
    const start = vi.fn();
    render(
      <IntegrationsOverview
        groups={groups(
          item({
            id: "bridge",
            name: "Remote terminals",
            status: "off",
            problem: { title: "Remote terminals are off", description: "Start Bridge to use them." },
            action: { label: "Start Bridge", onRun: start },
          }),
        )}
      />,
    );

    const attention = document.querySelector("[data-rcl-integrations-attention]");
    expect(attention).not.toBeNull();
    expect(within(attention as HTMLElement).getByText("Remote terminals are off")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Start Bridge" }));
    expect(start).toHaveBeenCalledTimes(1);
  });

  it("offers no fix for an item that is ready, even when it has one", () => {
    render(
      <IntegrationsOverview
        groups={groups(item({ id: "voice", name: "Voice", action: { label: "Start Audio Tools", onRun: vi.fn() }, inlineAction: true, problem: { title: "Voice is off" } }))}
      />,
    );
    expect(screen.queryByRole("button", { name: "Start Audio Tools" })).not.toBeInTheDocument();
    expect(document.querySelector("[data-rcl-integrations-attention]")).toBeNull();
  });

  it("puts an install action in the row for an optional tool that is missing", () => {
    const install = vi.fn();
    render(<IntegrationsOverview groups={groups(item({ id: "grok", name: "Grok", status: "missing", inlineAction: true, action: { label: "Install", onRun: install } }))} />);
    fireEvent.click(screen.getByRole("button", { name: "Install" }));
    expect(install).toHaveBeenCalledTimes(1);
  });

  it("shows a pending action as busy", () => {
    render(
      <IntegrationsOverview
        groups={groups(item({ id: "b", name: "Bridge", status: "off", problem: { title: "Off" }, action: { label: "Start Bridge", pendingLabel: "Starting…", pending: true, onRun: vi.fn() } }))}
      />,
    );
    expect(screen.getByRole("button", { name: "Starting…" })).toBeDisabled();
  });

  it("opens a detail view from a row and returns focus to that row on Back", () => {
    render(
      <IntegrationsOverview
        groups={groups(
          item({
            id: "voice",
            name: "Voice",
            provider: "Audio Tools",
            features: [
              { id: "dictation", label: "Dictate into the composer" },
              { id: "wake", label: "Wake word", available: false },
            ],
            facts: [{ label: "ID", value: "audio-tools", monospace: true }],
          }),
        )}
      />,
    );

    // A tap in Safari does not focus the button, so nothing is focused here.
    fireEvent.click(screen.getByRole("button", { name: "Voice" }));

    const heading = screen.getByRole("heading", { name: "Voice" });
    expect(heading).toHaveFocus();
    expect(screen.getByText("Provided by Audio Tools")).toBeInTheDocument();
    expect(screen.getByText("Dictate into the composer")).toBeInTheDocument();
    expect(screen.getByText("Unavailable")).toBeInTheDocument();
    expect(screen.getByText("audio-tools")).toHaveAttribute("data-monospace");

    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect(screen.getByRole("button", { name: "Voice" })).toHaveFocus();
  });

  it("does not make a row openable when there is nothing more to show", () => {
    render(<IntegrationsOverview groups={groups(item({ id: "claude", name: "Claude Code", subtitle: "Version 2.1" }))} />);
    expect(screen.queryByRole("button", { name: "Claude Code" })).not.toBeInTheDocument();
    expect(screen.getByText("Claude Code")).toBeInTheDocument();
  });

  it("lets the host control which detail view is open", () => {
    const onChange = vi.fn();
    render(<IntegrationsOverview groups={groups(item({ id: "voice", name: "Voice", facts: [{ label: "ID", value: "x" }] }))} selectedId="voice" onSelectedIdChange={onChange} />);
    expect(screen.getByRole("heading", { name: "Voice" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect(onChange).toHaveBeenCalledWith(null);
  });

  it("reports a failed check and offers to check again", () => {
    const refresh = vi.fn();
    render(<IntegrationsOverview groups={groups()} error="The service did not answer." onRefresh={refresh} />);
    expect(summary()).toBe("Couldn't check what's connected");
    expect(screen.getByText("The service did not answer.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Check again" }));
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("shows the first check as in progress", () => {
    render(<IntegrationsOverview groups={groups()} loading onRefresh={vi.fn()} />);
    expect(summary()).toBe("Checking…");
    expect(screen.getByRole("button", { name: "Checking…" })).toBeDisabled();
  });

  it("explains an empty group", () => {
    render(<IntegrationsOverview groups={[{ id: "accounts", label: "Accounts", items: [], emptyText: "No accounts connected yet." }]} />);
    expect(screen.getByText("No accounts connected yet.")).toBeInTheDocument();
  });

  it("uses translated strings", () => {
    render(<IntegrationsOverview groups={groups(item({ id: "a", name: "Voice" }))} strings={{ summaryAllReady: "{total} 件すべて利用可能", statusReady: "利用可能" }} />);
    expect(summary()).toBe("1 件すべて利用可能");
    expect(screen.getByText("利用可能")).toBeInTheDocument();
  });
});
