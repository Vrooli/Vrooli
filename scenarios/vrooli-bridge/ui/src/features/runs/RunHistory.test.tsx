/**
 * [REQ:BRG-P1-005] Run history — the durable remote-execution surface renders
 * live job output, exposes downloadable artifacts, and shows progress + a
 * cancel control + an ETA for a long/in-flight run (never a frozen spinner).
 *
 * This path is required verbatim by the requirements tracker.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";

import { renderWithProviders } from "../../test-utils";
import { RunStatus } from "../../api/runs";
import { RunEventKind } from "@vrooli/proto-types/vrooli-bridge/v1/channel/channel_pb";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { makeRun, makeRunEvent } from "./mocks/factories";

const { listRuns, getRun, abortRun } = vi.hoisted(() => ({
  listRuns: vi.fn(),
  getRun: vi.fn(),
  abortRun: vi.fn(),
}));

vi.mock("../../api/runs", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/runs")>();
  return { ...actual, runsClient: { listRuns, getRun, abortRun } };
});

import { RunHistory } from "./RunHistory";

describe("[REQ:BRG-P1-005] Run history", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows loading then the empty state", async () => {
    listRuns.mockReturnValue(new Promise(() => {}));
    const { unmount } = renderWithProviders(<RunHistory />);
    expect(screen.getByTestId(selectors.runs.loading)).toBeInTheDocument();
    unmount();
    cleanup();

    listRuns.mockResolvedValue({ runs: [] });
    renderWithProviders(<RunHistory />);
    await waitFor(() => expect(screen.getByTestId(selectors.runs.empty)).toBeInTheDocument());
  });

  it("surfaces a typed error when the feed fails", async () => {
    listRuns.mockRejectedValue(new ConnectError("denied", Code.Unavailable));
    renderWithProviders(<RunHistory />);
    await waitFor(() => expect(screen.getByTestId(selectors.runs.error)).toBeInTheDocument());
    expect(screen.getByText(strings.errors.unavailable)).toBeInTheDocument();
  });

  it("drills into a run and renders its live job output", async () => {
    const user = userEvent.setup();
    listRuns.mockResolvedValue({ runs: [makeRun({ id: "r1", status: RunStatus.PASSED })] });
    getRun.mockResolvedValue({
      run: makeRun({ id: "r1", status: RunStatus.PASSED }),
      events: [
        makeRunEvent({ runId: "r1", kind: RunEventKind.LOG, sequence: 1n, logChunk: "building module 42" }),
        makeRunEvent({ runId: "r1", kind: RunEventKind.EXIT, sequence: 2n, exitCode: 0 }),
      ],
    });
    renderWithProviders(<RunHistory />);

    await screen.findByTestId(selectors.runs.row({ id: "r1" }));
    await user.click(screen.getByTestId(selectors.runs.view({ id: "r1" })));

    const output = await screen.findByTestId(selectors.runs.output);
    await waitFor(() => expect(output).toHaveTextContent("building module 42"));
  });

  it("exposes downloadable artifacts for a run", async () => {
    const user = userEvent.setup();
    listRuns.mockResolvedValue({
      runs: [makeRun({ id: "r1", artifactRefs: ["dsh://bundle/abc"] })],
    });
    getRun.mockResolvedValue({
      run: makeRun({ id: "r1", artifactRefs: ["dsh://bundle/abc"] }),
      events: [],
    });
    renderWithProviders(<RunHistory />);

    await screen.findByTestId(selectors.runs.row({ id: "r1" }));
    await user.click(screen.getByTestId(selectors.runs.view({ id: "r1" })));

    const link = await screen.findByTestId(selectors.runs.artifact({ id: "r1", index: 0 }));
    // The artifact is rendered as an anchor with a download affordance and a
    // resolvable href.
    expect(link.tagName).toBe("A");
    expect(link).toHaveAttribute("download");
    expect(link.getAttribute("href")).toContain("dsh%3A%2F%2Fbundle%2Fabc");
    expect(link).toHaveTextContent("dsh://bundle/abc"); // ref shown as label
  });

  it("renders the empty-artifacts state when a run produced none", async () => {
    const user = userEvent.setup();
    listRuns.mockResolvedValue({ runs: [makeRun({ id: "r1", artifactRefs: [] })] });
    getRun.mockResolvedValue({ run: makeRun({ id: "r1", artifactRefs: [] }), events: [] });
    renderWithProviders(<RunHistory />);

    await screen.findByTestId(selectors.runs.row({ id: "r1" }));
    await user.click(screen.getByTestId(selectors.runs.view({ id: "r1" })));

    const artifacts = await screen.findByTestId(selectors.runs.artifacts);
    expect(within(artifacts).getByText(strings.runs.artifactsEmpty)).toBeInTheDocument();
  });

  it("shows progress + ETA + a cancel control for a long/in-flight run", async () => {
    const startedAgo = new Date(Date.now() - 30_000); // running for 30s
    listRuns.mockResolvedValue({
      runs: [
        makeRun({
          id: "running1",
          status: RunStatus.RUNNING,
          timeoutSeconds: 600n, // 10-minute budget → long job
          startedAt: timestampFromDate(startedAgo),
          finishedAt: undefined,
          exitCode: 0,
        }),
      ],
    });
    renderWithProviders(<RunHistory />);

    const row = await screen.findByTestId(selectors.runs.row({ id: "running1" }));
    const scoped = within(row);

    // Determinate progress bar (never a frozen spinner).
    const bar = scoped.getByRole("progressbar");
    expect(bar).toHaveAttribute("aria-valuenow");
    expect(Number(bar.getAttribute("aria-valuenow"))).toBeGreaterThanOrEqual(0);
    expect(Number(bar.getAttribute("aria-valuenow"))).toBeLessThan(100);

    // ETA is shown (a remaining-seconds estimate).
    expect(scoped.getByText(/~\d+s/)).toBeInTheDocument();

    // Cancel control is present for the in-flight run.
    expect(scoped.getByTestId(selectors.runs.cancel({ id: "running1" }))).toBeInTheDocument();
  });

  it("aborts a running run after confirmation", async () => {
    const user = userEvent.setup();
    listRuns.mockResolvedValue({
      runs: [
        makeRun({
          id: "running1",
          status: RunStatus.RUNNING,
          timeoutSeconds: 600n,
          startedAt: timestampFromDate(new Date(Date.now() - 5_000)),
          finishedAt: undefined,
        }),
      ],
    });
    abortRun.mockResolvedValue({});
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    renderWithProviders(<RunHistory />);

    await screen.findByTestId(selectors.runs.row({ id: "running1" }));
    await user.click(screen.getByTestId(selectors.runs.cancel({ id: "running1" })));

    expect(confirmSpy).toHaveBeenCalled();
    await waitFor(() => expect(abortRun).toHaveBeenCalledWith({ id: "running1" }));
    confirmSpy.mockRestore();
  });

  it("shows a terminal state (no spinner) for a finished run", async () => {
    listRuns.mockResolvedValue({
      runs: [makeRun({ id: "done1", status: RunStatus.PASSED, exitCode: 0 })],
    });
    renderWithProviders(<RunHistory />);

    const row = await screen.findByTestId(selectors.runs.row({ id: "done1" }));
    const scoped = within(row);
    // Terminal runs do not render the in-flight progress bar or cancel control.
    expect(scoped.queryByRole("progressbar")).not.toBeInTheDocument();
    expect(scoped.queryByTestId(selectors.runs.cancel({ id: "done1" }))).not.toBeInTheDocument();
  });
});
