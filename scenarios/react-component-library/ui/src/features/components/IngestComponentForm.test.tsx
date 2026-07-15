import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { selectors } from "../../consts/selectors";

vi.mock("../../api/components", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/components")>();
  return {
    ...actual,
    componentsClient: {
      ...actual.componentsClient,
      ingestComponent: vi.fn(),
    },
  };
});

const { getPromotionReadiness } = vi.hoisted(() => ({ getPromotionReadiness: vi.fn() }));
vi.mock("../../api/workflows", () => ({ workflowsClient: { getPromotionReadiness } }));

import { IngestComponentForm } from "./IngestComponentForm";
import { componentsClient } from "../../api/components";

const fillAndSubmit = async (user: ReturnType<typeof userEvent.setup>, opts: { acceptLoss?: boolean } = {}) => {
  await user.click(screen.getByTestId(selectors.components.ingest.details));
  await user.type(screen.getByTestId(selectors.components.ingest.scenario), "web-console");
  await user.type(screen.getByTestId(selectors.components.ingest.sourceFile), "ui/src/components/DrawerShell.tsx");
  await user.type(screen.getByTestId(selectors.components.ingest.slug), "DrawerShell");
  await user.type(screen.getByTestId(selectors.components.ingest.tags), " overlay , shell ,");
  if (opts.acceptLoss) {
    await user.click(screen.getByTestId(selectors.components.ingest.acceptLoss));
  }
  await user.click(screen.getByTestId(selectors.components.ingest.submit));
};

describe("IngestComponentForm", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("submits trimmed non-empty tags and surfaces the draft success message", async () => {
    vi.mocked(componentsClient.ingestComponent).mockResolvedValueOnce({
      draftVersion: "0.1.0",
      findings: [{ code: "x" }, { code: "y" }],
      parityReport: { acknowledged: false, findings: [] },
    } as never);

    const user = userEvent.setup();
    renderWithProviders(<IngestComponentForm />);
    await fillAndSubmit(user);

    await waitFor(() => {
      expect(componentsClient.ingestComponent).toHaveBeenCalledWith({
        scenario: "web-console",
        sourceFile: "ui/src/components/DrawerShell.tsx",
        slug: "DrawerShell",
        tags: ["overlay", "shell"],
        acceptBehaviorLoss: false,
      });
    });
    const success = await screen.findByTestId(selectors.components.ingest.success);
    expect(success.textContent).toContain("0.1.0");
    expect(success.textContent).toContain("2");
  });

  it("forwards the accept-behavior-loss override and reports accepted losses", async () => {
    vi.mocked(componentsClient.ingestComponent).mockResolvedValueOnce({
      draftVersion: "0.2.0",
      findings: [{ code: "x" }],
      parityReport: { acknowledged: true, findings: [{ code: "loss-a" }, { code: "loss-b" }] },
    } as never);

    const user = userEvent.setup();
    renderWithProviders(<IngestComponentForm />);
    await fillAndSubmit(user, { acceptLoss: true });

    await waitFor(() => {
      expect(componentsClient.ingestComponent).toHaveBeenCalledWith(
        expect.objectContaining({ acceptBehaviorLoss: true }),
      );
    });
    const success = await screen.findByTestId(selectors.components.ingest.success);
    // The accepted-losses notice reports the two acknowledged findings.
    expect(success.textContent).toContain("2 accepted");
  });

  it("shows read-only promotion blockers after a successful ingest", async () => {
    vi.mocked(componentsClient.ingestComponent).mockResolvedValueOnce({
      component: { id: "drawer-shell" }, draftVersion: "1.0.0", findings: [], parityReport: { acknowledged: false, findings: [] },
    } as never);
    getPromotionReadiness.mockResolvedValueOnce({ readiness: { ready: false, availableExampleCount: 0, requiredExampleCount: 1, blockers: ["origin replacement drift is not clean"], nextValidationCommand: "vrooli scenario test react-component-library" } });

    const user = userEvent.setup();
    renderWithProviders(<IngestComponentForm />);
    await fillAndSubmit(user);

    expect(await screen.findByText("Promotion evidence is incomplete.")).toBeInTheDocument();
    expect(screen.getByText("origin replacement drift is not clean")).toBeInTheDocument();
    expect(screen.getByText("vrooli scenario test react-component-library")).toBeInTheDocument();
    expect(getPromotionReadiness).toHaveBeenCalledWith({ assetId: "drawer-shell", originScenario: "web-console", version: "1.0.0" });
  });

  it("skips the accepted-losses notice when the response has no parity report", async () => {
    // No parityReport at all — exercises the optional-chaining branch where the
    // accepted-losses notice is skipped because there is no report.
    vi.mocked(componentsClient.ingestComponent).mockResolvedValueOnce({
      draftVersion: "0.3.0",
      findings: [{ code: "x" }],
    } as never);

    const user = userEvent.setup();
    renderWithProviders(<IngestComponentForm />);
    await fillAndSubmit(user);

    const success = await screen.findByTestId(selectors.components.ingest.success);
    expect(success.textContent).toContain("0.3.0");
    expect(success.textContent).not.toContain("accepted");
  });

  it("renders the error message when ingest fails", async () => {
    vi.mocked(componentsClient.ingestComponent).mockRejectedValueOnce(new Error("source file not found"));

    const user = userEvent.setup();
    renderWithProviders(<IngestComponentForm />);
    await fillAndSubmit(user);

    const error = await screen.findByTestId(selectors.components.ingest.error);
    expect(within(error).getByText(/source file not found/)).toBeInTheDocument();
  });
});
