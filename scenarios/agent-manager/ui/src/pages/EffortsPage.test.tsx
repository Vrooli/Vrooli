import { create } from "@bufbuild/protobuf";
import { fireEvent, screen, within } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { EffortBoardSchema, EffortFreshness } from "@vrooli/proto-types/agent-manager/v1/domain/effort_pb";
import { EffortsPage } from "./EffortsPage";
import { renderWithProviders } from "../test-utils";

const owner = vi.hoisted(() => ({ data: undefined as unknown, isPending: false, isFetching: false, error: null as Error | null, refetch: vi.fn(), tokens: [] as string[], refs: [] as string[] }));
vi.mock("../features/efforts/api", () => ({ useEffortBoard: (token: string, ref = "") => { owner.tokens.push(token); owner.refs.push(ref); return owner; } }));
const board = () => create(EffortBoardSchema, {
  activeCount: 2,
  partial: true,
  discovery: { generation: 3n, scannedCount: 2, scanLimit: 200, partial: true, findings: [{ code: "source_unavailable", source: "runtime", reason: "Owner unavailable" }] },
  rows: [
    { enrollment: { effortRef: "new-effort", displayName: "New arbitrary effort", revision: 2n, workShape: "adaptive-mandate" }, runtimeState: "running", outcomeStanding: { state: "unverified", attribution: "workspace self-report", limitations: ["No independent acceptance evidence"] }, freshness: EffortFreshness.STALE, usage: { partial: true, declaredRuns: 1 }, assignments: [{ subject: { role: "worker", runId: "run-one" }, requestedModel: "requested-model", effectiveModel: "effective-model", usage: { tokens: 0n, reportedCostUsd: 0, partial: false } }] },
    { enrollment: { effortRef: "other", displayName: "Finished runtime" }, runtimeState: "completed", outcomeStanding: { state: "unaccepted", attribution: "owner" }, freshness: EffortFreshness.FRESH },
  ],
});
beforeEach(() => { owner.data = undefined; owner.isPending = false; owner.isFetching = false; owner.error = null; owner.refetch.mockReset(); owner.tokens = []; owner.refs = []; });

test("an encoded exact link selects the intended row and does not fall back to another effort", () => {
  const exactRef = "effort:alpha/beta?x=1&second=2";
  const data = board(); data.rows[1].enrollment!.effortRef = exactRef; owner.data = data;
  renderWithProviders(<EffortsPage />, { initialEntries: [`/efforts?${new URLSearchParams({ effortRef: exactRef })}`] });
  expect(owner.refs.at(-1)).toBe(exactRef);
  expect(screen.getByRole("article", { name: `Effort ${exactRef} details` })).toBeInTheDocument();
  expect(screen.queryByRole("article", { name: "Effort new-effort details" })).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Show all efforts" }));
  expect(owner.refs.at(-1)).toBe("");
});

test("a missing exact effort remains unavailable even when other rows are returned", () => {
  owner.data = board();
  renderWithProviders(<EffortsPage />, { initialEntries: ["/efforts?effortRef=missing"] });
  expect(screen.getByRole("alert")).toHaveTextContent("no other effort has been selected");
  expect(screen.queryByRole("article")).toBeNull();
});

test.each([
  ["discovered", {}, /Not established; discovery is observation only/],
  ["enrolled", { authorizedBy: "operator" }, /Enrolled by operator/],
  ["withdrawn", { authorizedBy: "operator", withdrawn: true, withdrawalReason: "closed" }, /Withdrawn: closed/],
])("%s enrollment shows who owns it", (_state, fields, text) => {
  const data = board();
  Object.assign(data.rows[0].enrollment!, fields);
  owner.data = data;
  renderWithProviders(<EffortsPage />);
  expect(within(screen.getByRole("article", { name: "Effort new-effort details" })).getByText(text)).toBeInTheDocument();
});

test("loading and outage never assert that there are zero efforts", () => {
  owner.isPending = true;
  owner.isFetching = true;
  const view = renderWithProviders(<EffortsPage />);
  expect(screen.getByRole("status")).toHaveTextContent("Loading effort observation");
  expect(screen.getByRole("button", { name: "Refresh observation" })).toBeDisabled();
  owner.isPending = false; owner.isFetching = false; owner.error = new Error("offline");
  view.rerender(<EffortsPage />);
  expect(screen.getByRole("alert")).toHaveTextContent("Effort count and state are unknown");
  expect(screen.queryByText(/No enrollments/)).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Refresh observation" }));
  expect(owner.refetch).toHaveBeenCalledOnce();
});

test("arbitrary mixed efforts keep runtime, acceptance, stale evidence, model and unknown usage separate", () => {
  owner.data = board();
  renderWithProviders(<EffortsPage />);
  const detail = within(screen.getByRole("article", { name: "Effort new-effort details" }));
  expect(detail.getByText("unverified · workspace self-report")).toBeTruthy();
  expect(detail.getByText("Not established; discovery is observation only")).toBeTruthy();
  expect(detail.getByText("stale · unknown")).toBeTruthy();
  expect(detail.getByText("requested-model")).toBeTruthy();
  expect(detail.getByText("effective-model")).toBeTruthy();
  expect(detail.getByText("0.0000")).toBeTruthy();
  expect(detail.getAllByText("unknown").length).toBeGreaterThan(2);
  expect(detail.getByText("No independent acceptance evidence")).toBeTruthy();
  expect(detail.getByRole("link", { name: "Run run-one" })).toHaveAttribute("href", "/runs/run-one");
  expect(screen.getByRole("region", { name: "Discovery coverage" })).toHaveTextContent("source_unavailable");
  fireEvent.click(screen.getByRole("button", { name: /Finished runtime/ }));
  expect(screen.getByRole("article", { name: "Effort other details" })).toHaveTextContent("unaccepted · owner");
});

test("pagination is owner-cursor based and stale refresh errors retain explicit uncertainty", () => {
  const data = board(); data.nextPageToken = "opaque-owner-cursor"; owner.data = data;
  const view = renderWithProviders(<EffortsPage />);
  fireEvent.click(screen.getByRole("button", { name: "Next" }));
  expect(owner.tokens.at(-1)).toBe("opaque-owner-cursor");
  expect(screen.getByText("Page 2")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Previous" }));
  expect(owner.tokens.at(-1)).toBe("");
  owner.error = new Error("timeout"); view.rerender(<EffortsPage />);
  expect(screen.getByRole("alert")).toHaveTextContent("Retained results may be stale");
});

test("an empty observation retains coverage and makes no global no-work claim", () => {
  owner.data = create(EffortBoardSchema, { partial: true });
  renderWithProviders(<EffortsPage />);
  expect(screen.getByText(/Check discovery coverage before concluding/)).toBeTruthy();
  expect(screen.getByText("Discovery coverage unknown")).toBeTruthy();
  expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();
});
