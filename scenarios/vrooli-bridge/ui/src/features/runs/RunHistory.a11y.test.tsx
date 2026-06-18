/**
 * RunHistory accessibility regression tests.
 *
 * The runs feature owns its query-backed loading/success/empty UI plus the
 * in-flight progress bar, so a11y coverage lives here. Run status is conveyed
 * by icon + text (never color alone) and the progress bar carries an explicit
 * `role="progressbar"` with aria-value* — the populated assertions exercise
 * both a terminal run and a long in-flight run.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { RunStatus } from "../../api/runs";
import { makeRun } from "./mocks/factories";

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

describe("RunHistory accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders a populated feed (terminal + in-flight) without axe violations", async () => {
    listRuns.mockResolvedValue({
      runs: [
        makeRun({ id: "done1", status: RunStatus.PASSED, exitCode: 0 }),
        makeRun({
          id: "running1",
          status: RunStatus.RUNNING,
          timeoutSeconds: 600n,
          startedAt: timestampFromDate(new Date(Date.now() - 10_000)),
          finishedAt: undefined,
        }),
      ],
    });
    const { container } = renderWithProviders(<RunHistory />);

    await waitFor(() => expect(screen.getByTestId(selectors.runs.list)).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });

  it("renders the empty state without axe violations", async () => {
    listRuns.mockResolvedValue({ runs: [] });
    const { container } = renderWithProviders(<RunHistory />);

    await waitFor(() => expect(screen.getByTestId(selectors.runs.empty)).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });
});
