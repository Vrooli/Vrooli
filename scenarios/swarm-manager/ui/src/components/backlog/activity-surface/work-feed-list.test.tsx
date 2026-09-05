import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../../../test-utils";
import { WorkFeedList, type WorkFeedEntry } from "./work-feed-list";

const get = vi.fn();
vi.mock("../../../lib/api-client", () => ({ defaultApiClient: { get: (...args: unknown[]) => get(...args) } }));

const entries: WorkFeedEntry[] = [
  { id: "review/1", kind: "review", title: "Reviewing result", outcome: "accepted", actor: "reviewer", started_at: "2026-07-22T12:00:00Z", ended_at: "2026-07-22T12:03:00Z", detail_ref: "/reviews/1", detail_api_ref: "/api/v1/reviews/1", correlation: { execution_id: "execution-1" } },
  { id: "execution/1", kind: "execution", title: "Executing plan", outcome: "completed", actor: "operator", started_at: "2026-07-22T11:00:00Z", detail_ref: "/executions/execution-1" },
];

describe("WorkFeedList", () => {
  it("filters work episodes and opens typed episode context in a drawer", async () => {
    get.mockResolvedValue({ status: "accepted", agent_assessment: "Evidence is sufficient." });
    renderWithProviders(<WorkFeedList entries={entries} loading={false} error={null} />);
    expect(screen.getByText("Reviewing result")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Filter work history"), { target: { value: "review" } });
    expect(screen.getByText("Reviewing result")).toBeInTheDocument();
    expect(screen.queryByText("Executing plan")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Reviewing result/ }));
    await waitFor(() => expect(get).toHaveBeenCalledWith("/api/v1/reviews/1"));
    expect(screen.getByText("Evidence is sufficient.")).toBeInTheDocument();
    expect(screen.getByText("Correlation")).toBeInTheDocument();
    expect(screen.getByText("execution_id: execution-1")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open source detail" })).toHaveAttribute("href", "/reviews/1");
  });

  it("filters active and finished episodes by outcome", () => {
    renderWithProviders(<WorkFeedList entries={entries} loading={false} error={null} />);
    fireEvent.change(screen.getByLabelText("Filter work outcome"), { target: { value: "active" } });
    expect(screen.getByText("Executing plan")).toBeInTheDocument();
    expect(screen.queryByText("Reviewing result")).not.toBeInTheDocument();
  });
});
