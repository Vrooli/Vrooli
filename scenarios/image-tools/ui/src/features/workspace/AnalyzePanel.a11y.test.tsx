/**
 * AnalyzePanel accessibility regression tests. The panel owns the analysis op
 * list, the run/install controls, and the structured result views (probe /
 * OCR / NSFW), so a11y coverage lives here. Covers the idle action list and a
 * rendered probe result — both must be axe-clean in English.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { AnalyzePanel } from "./AnalyzePanel";
import { useAnalyze } from "./useAnalyze";
import { makeAnalyzeClient } from "./mocks/analysis";

vi.mock("../../api/analysis", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/analysis")>();
  const { makeAnalysisMocks } = await import("./mocks/analysis");
  return { ...actual, ...makeAnalysisMocks() };
});

const PNG = new File(["bytes"], "in.png", { type: "image/png" });

function Harness() {
  const analyze = useAnalyze({ client: makeAnalyzeClient() });
  return <AnalyzePanel analyze={analyze} input={PNG} />;
}

describe("AnalyzePanel accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the analysis action list without axe violations", async () => {
    const { container } = renderWithProviders(<Harness />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.analyzeAction({ name: "probe" }))).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });

  it("renders a probe result without axe violations", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<Harness />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.analyze.run)).toBeInTheDocument();
    });
    await user.click(screen.getByTestId(selectors.workspace.analyze.run));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.workspace.analyze.probe)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });
});
