import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { Header } from "./Header";
import { SnapshotPanel } from "./SnapshotPanel";

vi.mock("./StatusGrid", () => ({ StatusGrid: () => <div>Status grid</div> }));

describe("Header and SnapshotPanel", () => {
  afterEach(cleanup);

  it("refreshes from the header and exposes its loading state", () => {
    const onRefresh = vi.fn();
    const { rerender } = renderWithProviders(<Header isInitialLoading onRefresh={onRefresh} isRefreshing={false} />);
    expect(screen.getByText("Loading")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Refresh data" }));
    expect(onRefresh).toHaveBeenCalledOnce();
    rerender(<Header isInitialLoading={false} onRefresh={onRefresh} isRefreshing />);
    expect(screen.queryByText("Loading")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Refresh data" })).toBeDisabled();
  });

  it("toggles snapshot detail and displays loaded or skeleton statistics", () => {
    const { rerender } = renderWithProviders(
      <SnapshotPanel
        heroStats={{ overall_score: 80, readiness_label: "ready", risk_score: 2, confidence: 90, vault_configured: 3, vault_total: 4, missing_secrets: 1 }}
        updatedAt="2026-07-23T00:00:00Z"
        isLoading={false}
      />
    );
    expect(screen.getByText("Overall 80%")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /System Snapshot/ }));
    expect(screen.getByText("Status grid")).toBeInTheDocument();
    expect(screen.getByText("Missing secrets")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /System Snapshot/ }));
    rerender(<SnapshotPanel isLoading />);
    expect(screen.getByRole("button", { name: /System Snapshot/ })).toBeInTheDocument();
  });
});
