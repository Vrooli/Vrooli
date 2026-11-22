/**
 * Routing smoke — for each canonical path (`/`, `/settings`) the
 * matching page selector is in the document. Page-internal behaviour is
 * exercised in per-page tests; this file's job is to assert the router config.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "./routes";
import { Providers } from "./providers";

const mocks = vi.hoisted(() => ({ listDrafts: vi.fn(), listDraftAttachments: vi.fn(), listPostTypes: vi.fn(), listProjects: vi.fn(), createProject: vi.fn(), listCampaigns: vi.fn(), listClaims: vi.fn(), listDraftClaims: vi.fn(), listClaimProposals: vi.fn(), getClaimCoverage: vi.fn(), listReviewRuns: vi.fn(), listPublishRecords: vi.fn(), getDraftMetrics: vi.fn(), listRemediations: vi.fn() }));
vi.mock("../api/artifacts", () => ({ artifactsClient: { listDrafts: mocks.listDrafts, listDraftAttachments: mocks.listDraftAttachments } }));
vi.mock("../api/posttypes", () => ({ posttypesClient: { listPostTypes: mocks.listPostTypes } }));
vi.mock("../api/projects", () => ({ projectsClient: { listProjects: mocks.listProjects, createProject: mocks.createProject } }));
vi.mock("../api/campaigns", () => ({ campaignsClient: { listCampaigns: mocks.listCampaigns } }));
vi.mock("../api/claims", () => ({ claimsClient: { listClaims: mocks.listClaims, listDraftClaims: mocks.listDraftClaims, listClaimProposals: mocks.listClaimProposals, getClaimCoverage: mocks.getClaimCoverage } }));
vi.mock("../api/review", () => ({ reviewClient: { listReviewRuns: mocks.listReviewRuns } }));
vi.mock("../api/ledger", () => ({ ledgerClient: { listPublishRecords: mocks.listPublishRecords, getDraftMetrics: mocks.getDraftMetrics, listRemediations: mocks.listRemediations } }));
vi.mock("../pages/ReviewPage", () => ({ ReviewPage: () => <div data-testid={selectors.pages.review} /> }));

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    mocks.listDrafts.mockResolvedValue({ drafts: [] });
    mocks.listDraftAttachments.mockResolvedValue({ attachments: [] });
    mocks.listPostTypes.mockResolvedValue({ postTypes: [] });
    mocks.listProjects.mockResolvedValue({ projects: [] });
    mocks.listCampaigns.mockResolvedValue({ campaigns: [] });
    mocks.listClaims.mockResolvedValue({ claims: [] });
    mocks.listDraftClaims.mockResolvedValue({ claims: [] });
    mocks.listClaimProposals.mockResolvedValue({ proposals: [] });
    mocks.getClaimCoverage.mockResolvedValue({});
    mocks.listReviewRuns.mockResolvedValue({ reviewRuns: [] });
    mocks.listPublishRecords.mockResolvedValue({ publishRecords: [] });
    mocks.getDraftMetrics.mockResolvedValue({ hasMeasurements: false, readings: [] });
    mocks.listRemediations.mockResolvedValue({ remediations: [] });
  });

  it("renders the dashboard at /", () => {
    renderWithProviders(<Providers><TestAppRouter initialEntries={["/"]} /></Providers>, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<Providers><TestAppRouter initialEntries={["/settings"]} /></Providers>, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });

  it("renders the additive Today workspace at /today", async () => {
    renderWithProviders(<Providers><TestAppRouter initialEntries={["/today"]} /></Providers>, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.today)).toBeInTheDocument();
  });

  it("renders the additive Plan workspace at /plan", async () => {
    renderWithProviders(<Providers><TestAppRouter initialEntries={["/plan"]} /></Providers>, { withoutRouter: true });
    expect(await screen.findByTestId("page-plan")).toBeInTheDocument();
  });

  it("renders the additive Create workspace at /create", async () => {
    renderWithProviders(<Providers><TestAppRouter initialEntries={["/create"]} /></Providers>, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.create)).toBeInTheDocument();
  });

  it("renders the exact-draft Review workbench at /review", async () => {
    renderWithProviders(<Providers><TestAppRouter initialEntries={["/review?draft=draft-to-review"]} /></Providers>, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.review)).toBeInTheDocument();
  });

  it("renders the read-only Learn view at /learn", async () => {
    renderWithProviders(<Providers><TestAppRouter initialEntries={["/learn"]} /></Providers>, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.learn)).toBeInTheDocument();
  });
});
