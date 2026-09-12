/**
 * Routing smoke — for each canonical path the matching page selector is in the
 * document. The drill-down detail routes are code-split (React.lazy), so their
 * assertions await the loaded page selector; the eager overview routes resolve
 * synchronously. Page-internal behaviour is exercised in per-page tests; this
 * file's job is to assert the router config wires each path to its page.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "./routes";

vi.mock("../api/templateDomain", () => ({
  fetchTemplateDashboard: vi.fn().mockResolvedValue({
    templates: { templates: [] },
    runs: { runs: [] },
    drift: { snapshots: [] },
    debt: { entries: [] },
    monitor: { status: { enabled: true, intervalSeconds: 0n, inFlight: false, lastStatus: "scheduled", lastRunId: "", greenStreak: 0n } },
  }),
  fetchTemplateList: vi.fn().mockResolvedValue([
    { id: "react-vite", kind: 1, displayName: "React + Vite", version: "1.6.0", status: "active", tags: [], manifestPath: "", sourcePath: "" },
  ]),
  fetchTemplateDetail: vi.fn().mockResolvedValue({
    template: { id: "react-vite", kind: 1, displayName: "React + Vite", version: "1.6.0", status: "active", tags: [], manifestPath: "", sourcePath: "" },
    runs: [],
    drift: [],
    debt: [],
  }),
  fetchValidationRunList: vi.fn().mockResolvedValue([
    { id: "validation-1", templateId: "react-vite", mode: 2, status: "passed", trigger: "monitor", findings: [] },
  ]),
  fetchValidationRun: vi.fn().mockResolvedValue({
    id: "validation-1", templateId: "react-vite", mode: 2, status: "passed", target: "fleet", trigger: "monitor", phaseResults: [], findings: [],
  }),
  fetchDebtEntry: vi.fn().mockResolvedValue({
    key: "react-vite.aria", templateId: "react-vite", severity: "medium", status: "open", title: "Missing aria label", source: "ui-health", detail: "",
  }),
  fetchDebtLedger: vi.fn().mockResolvedValue({ entries: [], templates: [] }),
}));

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the dashboard at /", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });

  it("renders the template list at /templates", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/templates"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.templateList)).toBeInTheDocument();
  });

  it("renders the run list at /runs", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/runs"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.runList)).toBeInTheDocument();
  });

  it("renders the debt list at /debt", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/debt"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.debtList)).toBeInTheDocument();
  });

  it("renders the template detail at /templates/:templateId", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/templates/react-vite"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.templateDetail)).toBeInTheDocument();
  });

  it("renders the run detail at /runs/:runId", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/runs/validation-1"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.runDetail)).toBeInTheDocument();
  });

  it("renders the debt detail at /debt/:debtKey", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/debt/react-vite.aria"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.debtDetail)).toBeInTheDocument();
  });
});
