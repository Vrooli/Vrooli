/**
 * Detail/editor surfaces — route params, query data, and primary actions.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router-dom";
import { create } from "@bufbuild/protobuf";

import {
  GoldenSchema,
  ListGoldensResponseSchema,
} from "@vrooli/proto-types/development-toolchain-validator/v1/golden/golden_pb";
import {
  GetGoldenSummaryResponseSchema,
  GetTupleHistoryResponseSchema,
  GoldenSummarySchema,
  TupleHistorySchema,
  TupleVerdictSchema,
} from "@vrooli/proto-types/development-toolchain-validator/v1/report/report_pb";
import {
  ClearStaleResponseSchema,
  ConvergenceTarget,
  ContentRuleSchema,
  GetManifestResponseSchema,
  ListManifestsResponseSchema,
  ManifestSchema,
  UpsertManifestResponseSchema,
} from "@vrooli/proto-types/development-toolchain-validator/v1/manifest/manifest_pb";
import {
  GetSkillResponseSchema,
  ListSkillsResponseSchema,
  SkillSchema,
} from "@vrooli/proto-types/development-toolchain-validator/v1/skill_catalog/skill_catalog_pb";
import {
  StartResponseSchema,
  ValidationRunSchema,
} from "@vrooli/proto-types/development-toolchain-validator/v1/validation_run/validation_run_pb";
import {
  TupleKind,
  ValidationRecordSchema,
  Verdict,
} from "@vrooli/proto-types/development-toolchain-validator/v1/validation_record/validation_record_pb";

import { selectors } from "../../consts/selectors";
import { ROUTE_PATTERNS } from "../../routes.generated";
import { renderWithProviders } from "../../test-utils";
import { GoldenDetail } from "../goldens/GoldenDetail";
import { TupleDetail } from "../goldens/TupleDetail";
import { ManifestEditor } from "../manifests/ManifestEditor";
import { ManifestsIndex } from "../manifests/ManifestsIndex";
import { SkillDetail } from "../skills/SkillDetail";
import { SkillsIndex } from "../skills/SkillsIndex";

vi.mock("../../api/golden", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/golden")>();
  return {
    ...actual,
    goldenClient: {
      listGoldens: vi.fn(),
      registerGolden: vi.fn(),
      regenerateGolden: vi.fn(),
      deleteGolden: vi.fn(),
    },
  };
});

vi.mock("../../api/report", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/report")>();
  return {
    ...actual,
    reportClient: {
      getGoldenSummary: vi.fn(),
      getTupleHistory: vi.fn(),
      getCoverage: vi.fn(),
      getSkillFitness: vi.fn(),
    },
  };
});

vi.mock("../../api/manifest", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/manifest")>();
  return {
    ...actual,
    manifestClient: {
      listManifests: vi.fn(),
      getManifest: vi.fn(),
      upsertManifest: vi.fn(),
      clearStale: vi.fn(),
    },
  };
});

vi.mock("../../api/skillCatalog", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/skillCatalog")>();
  return {
    ...actual,
    skillCatalogClient: {
      sync: vi.fn(),
      listSkills: vi.fn(),
      getSkill: vi.fn(),
    },
  };
});

vi.mock("../../api/validationRun", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/validationRun")>();
  return {
    ...actual,
    validationRunClient: {
      start: vi.fn(),
      get: vi.fn(),
      listActive: vi.fn(),
    },
  };
});

import { goldenClient } from "../../api/golden";
import { reportClient } from "../../api/report";
import { manifestClient } from "../../api/manifest";
import { skillCatalogClient } from "../../api/skillCatalog";
import { validationRunClient } from "../../api/validationRun";

const slug = "reference-react-vite";
const skillId = "progress";
const legacyCommittedPath = ["scenarios", "reference-react-vite"].join("/");

const golden = () =>
  create(GoldenSchema, {
    slug,
    templateId: "react-vite",
    templateVersionPinned: "1.4.0",
    path: ".vrooli/generated-goldens/reference-react-vite",
    logicalRoot: ".vrooli/generated-goldens/reference-react-vite",
  });

const manifest = () =>
  create(ManifestSchema, {
    skillId,
    goldenSlug: slug,
    allowedPaths: ["api/**", "ui/src/**"],
    wildcardAllowed: false,
    convergenceTarget: ConvergenceTarget.EMPTY_DIFF,
    contentRules: [
      create(ContentRuleSchema, {
        pathGlob: "api/**",
        mustContain: ["generated golden"],
        mustNotContain: [legacyCommittedPath],
      }),
    ],
    templateVersionPinned: "1.4.0",
    skillVersionPinned: "2026-07-02T00:00:00Z",
  });

beforeEach(() => {
  vi.mocked(goldenClient.listGoldens).mockResolvedValue(
    create(ListGoldensResponseSchema, { goldens: [golden()] }),
  );
  vi.mocked(goldenClient.regenerateGolden).mockResolvedValue({ golden: golden() });
  vi.mocked(goldenClient.deleteGolden).mockResolvedValue({});
  vi.mocked(reportClient.getGoldenSummary).mockResolvedValue(
    create(GetGoldenSummaryResponseSchema, {
      summary: create(GoldenSummarySchema, {
        goldenSlug: slug,
        skillVerdicts: [
          create(TupleVerdictSchema, {
            tupleKind: TupleKind.SKILL,
            subjectId: skillId,
            latestVerdict: Verdict.PASS,
          }),
        ],
        toolVerdicts: [
          create(TupleVerdictSchema, {
            tupleKind: TupleKind.TOOL,
            subjectId: "test-genie",
            latestVerdict: Verdict.RUN_FAILURE,
          }),
        ],
      }),
    }),
  );
  vi.mocked(reportClient.getTupleHistory).mockResolvedValue(
    create(GetTupleHistoryResponseSchema, {
      history: create(TupleHistorySchema, {
        tupleKind: TupleKind.SKILL,
        subjectId: skillId,
        goldenSlug: slug,
        records: [
          create(ValidationRecordSchema, {
            id: "record-1",
            subjectId: skillId,
            goldenSlug: slug,
            tupleKind: TupleKind.SKILL,
            verdict: Verdict.PASS,
            durationMs: 42,
            tokensUsed: 100,
            costUsdMicro: 20,
            diffHash: "diff-123",
            agentManagerRunId: "agent-run-1",
          }),
        ],
      }),
    }),
  );
  vi.mocked(manifestClient.listManifests).mockResolvedValue(
    create(ListManifestsResponseSchema, { manifests: [manifest()] }),
  );
  vi.mocked(manifestClient.getManifest).mockResolvedValue(
    create(GetManifestResponseSchema, { manifest: manifest() }),
  );
  vi.mocked(manifestClient.upsertManifest).mockResolvedValue(
    create(UpsertManifestResponseSchema, { manifest: manifest() }),
  );
  vi.mocked(manifestClient.clearStale).mockResolvedValue(
    create(ClearStaleResponseSchema, {}),
  );
  vi.mocked(skillCatalogClient.listSkills).mockResolvedValue(
    create(ListSkillsResponseSchema, {
      skills: [
        create(SkillSchema, {
          id: skillId,
          version: "2026-07-02T00:00:00Z",
          contentHash: "abc123",
        }),
      ],
    }),
  );
  vi.mocked(skillCatalogClient.getSkill).mockResolvedValue(
    create(GetSkillResponseSchema, {
      skill: create(SkillSchema, {
        id: skillId,
        version: "2026-07-02T00:00:00Z",
        contentHash: "abc123",
      }),
    }),
  );
  vi.mocked(validationRunClient.start).mockResolvedValue(
    create(StartResponseSchema, {
      run: create(ValidationRunSchema, { id: "run-1" }),
    }),
  );
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const routeRender = (pattern: string, entry: string, element: JSX.Element) =>
  renderWithProviders(
    <Routes>
      <Route path={pattern} element={element} />
      <Route path={ROUTE_PATTERNS.runDetail} element={<div data-testid="run-detail-target" />} />
      <Route path={ROUTE_PATTERNS.goldensIndex} element={<div data-testid="goldens-index-target" />} />
      <Route path={ROUTE_PATTERNS.manifestsIndex} element={<div data-testid="manifests-index-target" />} />
      <Route path={ROUTE_PATTERNS.skillsIndex} element={<div data-testid="skills-index-target" />} />
    </Routes>,
    { routerEntries: [entry] },
  );

describe("detail surfaces", () => {
  it("renders golden verdict grids and runs regenerate/delete actions", async () => {
    const user = userEvent.setup();
    routeRender(ROUTE_PATTERNS.goldenDetail, `/goldens/${slug}`, <GoldenDetail />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.goldens.detail)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.goldens.skillsGrid)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.goldens.toolsGrid)).toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.goldens.detailRegenerate));
    await user.click(
      within(screen.getByRole("dialog", { name: "goldens.confirmRegenerate" })).getByRole(
        "button",
        { name: "goldens.regenerate" },
      ),
    );
    await waitFor(() => {
      expect(goldenClient.regenerateGolden).toHaveBeenCalledWith({ slug });
    });

    await user.click(screen.getByTestId(selectors.goldens.detailDelete));
    await user.click(
      within(screen.getByRole("dialog", { name: "goldens.confirmDelete" })).getByRole(
        "button",
        { name: "goldens.delete" },
      ),
    );
    await waitFor(() => {
      expect(goldenClient.deleteGolden).toHaveBeenCalledWith({ slug });
    });
  });

  it("renders tuple history, manifest details, and starts a validation run", async () => {
    const user = userEvent.setup();
    routeRender(
      ROUTE_PATTERNS.tupleDetail,
      `/goldens/${slug}/skill/${skillId}`,
      <TupleDetail />,
    );

    await waitFor(() => {
      expect(screen.getByTestId(selectors.goldens.tupleDetail)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.goldens.tupleDetailRunSummary)).toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.goldens.tupleDetailManifest));
    await waitFor(() => {
      expect(manifestClient.getManifest).toHaveBeenCalledWith({ skillId, goldenSlug: slug });
    });

    await user.click(screen.getByTestId(selectors.goldens.tupleDetailHistory));
    expect(screen.getAllByText("PASS").length).toBeGreaterThan(0);

    await user.click(screen.getByTestId(selectors.runs.runValidation));
    await waitFor(() => {
      expect(validationRunClient.start).toHaveBeenCalledWith(
        expect.objectContaining({ tupleKind: TupleKind.SKILL, subjectId: skillId, goldenSlug: slug }),
      );
    });
  });

  it("renders tuple empty/tool branches without a skill manifest", async () => {
    const user = userEvent.setup();
    vi.mocked(reportClient.getTupleHistory).mockResolvedValueOnce(
      create(GetTupleHistoryResponseSchema, {
        history: create(TupleHistorySchema, {
          tupleKind: TupleKind.TOOL,
          subjectId: "test-genie",
          goldenSlug: slug,
          records: [],
        }),
      }),
    );
    routeRender(
      ROUTE_PATTERNS.tupleDetail,
      `/goldens/${slug}/tool/test-genie`,
      <TupleDetail />,
    );

    await waitFor(() => {
      expect(screen.getByTestId(selectors.goldens.tupleDetail)).toBeInTheDocument();
    });
    expect(screen.getByText("goldens.tupleNoRuns")).toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.goldens.tupleDetailManifest));
    expect(screen.getByText("goldens.tupleManifestEmpty")).toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.goldens.tupleDetailHistory));
    expect(screen.getByText("goldens.tupleHistoryEmpty")).toBeInTheDocument();
  });

  it("edits, validates, saves, and clears a manifest", async () => {
    const user = userEvent.setup();
    routeRender(
      ROUTE_PATTERNS.manifestEditor,
      `/manifests/${skillId}/${slug}`,
      <ManifestEditor />,
    );

    await waitFor(() => {
      expect(screen.getByTestId(selectors.manifests.editor)).toBeInTheDocument();
    });

    const contentRules = screen.getByTestId(selectors.manifests.editorContentRules);
    await user.clear(contentRules);
    await user.type(contentRules, "not json");
    await user.click(screen.getByTestId(selectors.manifests.editorSave));
    expect(await screen.findByText(/Unexpected token|invalid/i)).toBeInTheDocument();

    await user.clear(contentRules);
    fireEvent.change(contentRules, {
      target: { value: '[{"pathGlob":"ui/**","mustContain":["generated"]}]' },
    });
    await user.click(screen.getByTestId(selectors.manifests.editorWildcardAllowed));
    await user.click(screen.getByTestId(selectors.manifests.editorSave));
    await waitFor(() => {
      expect(manifestClient.upsertManifest).toHaveBeenCalled();
    });

    await user.click(screen.getByTestId(selectors.manifests.editorClearStale));
    await waitFor(() => {
      expect(manifestClient.clearStale).toHaveBeenCalledWith({ skillId, goldenSlug: slug });
    });
  });

  it("renders manifest and skill index rows plus skill detail", async () => {
    routeRender(ROUTE_PATTERNS.manifestsIndex, "/manifests", <ManifestsIndex />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.manifests.row)).toBeInTheDocument();
    });

    cleanup();
    routeRender(ROUTE_PATTERNS.skillsIndex, "/skills", <SkillsIndex />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.skills.row)).toBeInTheDocument();
    });

    cleanup();
    routeRender(ROUTE_PATTERNS.skillDetail, `/skills/${skillId}`, <SkillDetail />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.skills.detail)).toBeInTheDocument();
    });
    expect(screen.getByText("abc123")).toBeInTheDocument();
  });

  it("renders empty and error states for detail indexes", async () => {
    vi.mocked(goldenClient.listGoldens).mockResolvedValueOnce(
      create(ListGoldensResponseSchema, { goldens: [] }),
    );
    routeRender(ROUTE_PATTERNS.goldenDetail, "/goldens/missing", <GoldenDetail />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.goldens.empty)).toBeInTheDocument();
    });

    cleanup();
    vi.mocked(skillCatalogClient.getSkill).mockResolvedValueOnce(
      create(GetSkillResponseSchema, {}),
    );
    routeRender(ROUTE_PATTERNS.skillDetail, `/skills/${skillId}`, <SkillDetail />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.skills.empty)).toBeInTheDocument();
    });

    cleanup();
    vi.mocked(skillCatalogClient.listSkills).mockRejectedValueOnce(new Error("boom"));
    routeRender(ROUTE_PATTERNS.skillsIndex, "/skills", <SkillsIndex />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.skills.error)).toBeInTheDocument();
    });
  });
});
