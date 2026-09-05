import { create } from "@bufbuild/protobuf";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { CohortWatchSchema, InspectCohortWatchResponseSchema, WatchDisposition, WatchStatus } from "@vrooli/proto-types/agent-manager/v1/domain/watch_pb";
import { WatchesPage } from "../../src/pages/WatchesPage";
import { renderWithProviders } from "../../src/test-utils";
import type { CohortWatchInspection } from "../../src/hooks/useApi";

const owner = vi.hoisted(() => ({ data: [] as unknown[], loading: false, error: null as string | null, inspect: vi.fn(), refetch: vi.fn() }));
vi.mock("../../src/hooks/useApi", () => ({ useCohortWatches: () => owner }));
const watch = (id: string) => create(CohortWatchSchema, { watchId: id, status: WatchStatus.ACTIVE, revision: 2n, spec: { familyExecutionId: "family", policyVersion: "policy-1", subjects: [{ runId: "child" }] } });
const detail = (id: string): CohortWatchInspection => ({ inspection: create(InspectCohortWatchResponseSchema, { watch: watch(id) }), actions: [] });
beforeEach(() => { owner.data = []; owner.loading = false; owner.error = null; owner.inspect.mockReset(); owner.refetch.mockReset(); });

test("watch console exposes loading, failure, empty state and refresh", () => {
  owner.loading = true;
  const view = renderWithProviders(<WatchesPage />);
  expect(screen.getByText("Loading cohort watches…")).toBeTruthy();
  expect(screen.getByRole("button", { name: "Refresh" })).toBeDisabled();
  owner.loading = false;
  owner.error = "Owner unavailable";
  view.rerender(<WatchesPage />);
  expect(screen.getByRole("alert")).toHaveTextContent("Owner unavailable");
  expect(screen.getByText("No cohort watches yet.")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
  expect(owner.refetch).toHaveBeenCalledOnce();
});

test("watch console keeps selection authoritative when an earlier inspection finishes late", async () => {
  owner.data = [watch("first-watch"), watch("second-watch")];
  let finishFirst!: (value: CohortWatchInspection) => void;
  owner.inspect.mockImplementation((id: string) => id === "first-watch" ? new Promise<CohortWatchInspection>(resolve => { finishFirst = resolve; }) : Promise.resolve(detail(id)));
  renderWithProviders(<WatchesPage />);
  fireEvent.click(screen.getByRole("button", { name: /second-w/ }));
  await waitFor(() => expect(screen.getByRole("article", { name: "Watch second-watch details" })).toBeTruthy());
  await act(async () => finishFirst(detail("first-watch")));
  expect(screen.queryByRole("article", { name: "Watch first-watch details" })).toBeNull();
  expect(screen.getByText("No decision recorded.")).toBeTruthy();
  expect(screen.getByText("No actions recorded.")).toBeTruthy();
});

test("watch console reports cursor repair and accepted or rejected action evidence", async () => {
  const result = detail("evidence-watch");
  result.inspection.cursorResetRequired = true;
  result.inspection.events = [{ eventId: "event-1", runId: "child-1", eventType: "terminal", sequence: 9n } as typeof result.inspection.events[number]];
  result.inspection.watch = create(CohortWatchSchema, { ...watch("evidence-watch"), lastDecision: { disposition: WatchDisposition.SIGNAL, classification: "friction", confidence: 0.8, evidenceIds: ["event-1"], recommendedAction: "wake_parent" } });
  result.actions = [
    { actionId: "a", kind: 1, targetRunId: "child-1", state: 2, status: "", rejectionReason: "" },
    { actionId: "b", kind: 1, targetRunId: "", state: 4, status: "", rejectionReason: "Policy limit" },
    { actionId: "c", kind: 1, targetRunId: "", state: 99, status: "delayed", rejectionReason: "" },
  ];
  owner.data = [watch("evidence-watch")];
  owner.inspect.mockResolvedValue(result);
  renderWithProviders(<WatchesPage />);
  await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Cursor reset required: retention changed"));
  expect(screen.getByText(/signal · friction · confidence 0.80/)).toBeTruthy();
  expect(screen.getByText("#9")).toBeTruthy();
  expect(screen.getByText("1 · accepted")).toBeTruthy();
  expect(screen.getByText("Policy limit")).toBeTruthy();
  expect(screen.getByText("1 · delayed")).toBeTruthy();
});

test.each([new Error("Inspection failed"), "untyped failure"])("watch console retains owner inspection errors", async error => {
  owner.data = [watch("failed-watch")];
  owner.inspect.mockRejectedValue(error);
  renderWithProviders(<WatchesPage />);
  await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent(error instanceof Error ? error.message : "Failed to inspect cohort watch"));
});

test("watch console refuses evidence attributed to another watch", async () => {
  owner.data = [watch("selected-watch")];
  owner.inspect.mockResolvedValue(detail("unrelated-watch"));
  renderWithProviders(<WatchesPage />);
  await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Inspection returned a different watch"));
  expect(screen.queryByRole("article", { name: "Watch unrelated-watch details" })).toBeNull();
});
