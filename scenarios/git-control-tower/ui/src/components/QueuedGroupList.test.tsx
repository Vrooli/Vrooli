import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { StagedMessageMode } from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";
import { QueuedGroupList } from "./QueuedGroupList";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import type { QueuedStageSelection } from "../features/commit-message/stageQueue";

const queued = (id: string, paths: string[], patch: Partial<QueuedStageSelection> = {}): QueuedStageSelection => ({
  id,
  paths,
  queuedAt: "2026-11-24T12:00:00.000Z",
  state: "queued",
  selectionId: id,
  snapshotDigest: `sha256:${id}`,
  ...patch,
});

describe("QueuedGroupList", () => {
  it("summarizes groups and starts the first available one", () => {
    const start = vi.fn();
    const first = queued("a", ["a.ts", "b.ts"]);
    renderWithProviders(<QueuedGroupList repositoryId="repo" selections={[first, queued("b", ["c.ts"])]} onUse={vi.fn()} onStartSuggestion={start} />);

    expect(screen.getByTestId("queued-stage-selections")).toHaveTextContent("2 groups · 3 files");
    fireEvent.click(screen.getByTestId("start-next-queued-group"));
    expect(start).toHaveBeenCalledWith(first, StagedMessageMode.QUICK);
  });

  it("keeps the displaced commit action explicit and compact", () => {
    const resolve = vi.fn();
    renderWithProviders(<QueuedGroupList repositoryId="repo" selections={[queued("a", ["a.ts"], { state: "commit_pending" })]} onUse={vi.fn()} onResolveCommit={resolve} />);

    fireEvent.click(screen.getByRole("button", { name: "I committed these files" }));
    expect(resolve).toHaveBeenCalledWith("a", true);
  });

  it("uses the row itself as the expansion control", () => {
    renderWithProviders(<QueuedGroupList repositoryId="repo" selections={[queued("a", ["a.ts"])]} onUse={vi.fn()} />);
    const row = screen.getByTestId("queued-group-details");
    expect(row).toHaveAttribute("aria-expanded", "false");
    fireEvent.click(row);
    expect(row).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByTestId("queued-suggestion-card")).toBeInTheDocument();
  });

  it("shows an in-flight start instead of accepting a second click", () => {
    const start = vi.fn();
    renderWithProviders(<QueuedGroupList repositoryId="repo" selections={[queued("a", ["a.ts"])]} onUse={vi.fn()} onStartSuggestion={start} startingIds={["a"]} />);
    const button = screen.getByTestId("start-next-queued-group");
    expect(button).toHaveTextContent("Starting…");
    expect(button).toBeDisabled();
    fireEvent.click(button);
    expect(start).not.toHaveBeenCalled();
  });

  it("collapses the verbose card from its own header", () => {
    renderWithProviders(<QueuedGroupList repositoryId="repo" selections={[queued("a", ["a.ts"])]} onUse={vi.fn()} />);
    fireEvent.click(screen.getByTestId("queued-group-details"));
    expect(screen.getByTestId("queued-suggestion-card")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("queued-suggestion-collapse"));
    expect(screen.queryByTestId("queued-suggestion-card")).not.toBeInTheDocument();
    expect(screen.getByTestId("queued-group-details")).toHaveAttribute("aria-expanded", "false");
  });

  it("lets a stale group be removed from the compact row", () => {
    const remove = vi.fn();
    renderWithProviders(<QueuedGroupList repositoryId="repo" selections={[queued("a", ["a.ts"], { state: "stale" })]} onUse={vi.fn()} onRemove={remove} />);
    fireEvent.click(screen.getByRole("button", { name: "Remove group 1" }));
    expect(remove).toHaveBeenCalledWith("a");
  });

  it("marks a group briefly revealed by the blocker action", () => {
    renderWithProviders(<QueuedGroupList repositoryId="repo" selections={[queued("a", ["a.ts"])]} onUse={vi.fn()} revealedId="a" />);
    expect(screen.getByTestId("queued-group-row")).toHaveAttribute("data-revealed", "true");
  });

  it("does not render the verbose card until details are requested", () => {
    renderWithProviders(<QueuedGroupList repositoryId="repo" selections={[queued("a", ["a.ts"])]} onUse={vi.fn()} />);
    expect(screen.queryByTestId("queued-suggestion-card")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("queued-group-details"));
    expect(screen.getByTestId("queued-suggestion-card")).toBeInTheDocument();
  });
});
