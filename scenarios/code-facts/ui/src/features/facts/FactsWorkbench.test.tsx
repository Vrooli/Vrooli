import { create } from "@bufbuild/protobuf";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../../consts/selectors";
import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { FactsWorkbench } from "./FactsWorkbench";
import {
  CacheMetadataSchema,
  CodeFactsReportSchema,
  EvidenceSchema,
  EvidenceStatus,
  FactFamily,
  GenericFactSchema,
  IndexStatusSchema,
  IndexJobSchema,
  ParseUnitSchema,
  SourceRangeSchema,
  SearchExpansionSchema,
  SearchHitSchema,
  SearchRankFactorSchema,
  SearchResponseSchema,
  SurfaceKind,
  SurfaceSchema,
  SurfaceStatus,
  TargetContextSchema,
  TargetKind,
  WarningSchema,
} from "@vrooli/proto-types/code-facts/v1/facts/facts_pb";

vi.mock("../../api/facts", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/facts")>();
  return {
    ...actual,
    describeCodeFacts: vi.fn(),
    getIndexStatus: vi.fn(),
    searchCodeFacts: vi.fn(),
  };
});

const { describeCodeFacts, getIndexStatus, searchCodeFacts } = await import("../../api/facts");

describe("FactsWorkbench", () => {
  beforeEach(() => {
    vi.mocked(describeCodeFacts).mockReset();
    vi.mocked(getIndexStatus).mockReset();
    vi.mocked(searchCodeFacts).mockReset();
    vi.mocked(getIndexStatus).mockResolvedValue(create(IndexStatusSchema, {
      activeGeneration: "gen-active",
      state: "ready",
      sourceFiles: 42n,
      searchDocuments: 128n,
      semanticCards: 64n,
      graphFacts: 256n,
      storageBytes: 1048576n,
    }));
  });

  it("renders an empty operator console before analysis runs", () => {
    renderWithProviders(<FactsWorkbench />);

    expect(screen.getByTestId(selectors.facts.workbench)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.facts.empty)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.facts.targetInput)).toHaveValue("code-facts");
    expect(screen.getByTestId(selectors.facts.familyControls)).toBeInTheDocument();
  });

  it("keeps the evidence workspace free of automated accessibility violations", async () => {
    const { container } = renderWithProviders(<FactsWorkbench />);
    await screen.findByText("Index ready");
    await expectNoA11yViolations(container);
  });

  it("searches ranked evidence and exposes trust and relationship detail", async () => {
    const user = userEvent.setup();
    vi.mocked(searchCodeFacts).mockResolvedValueOnce(create(SearchResponseSchema, {
      generation: "gen-active",
      retrievalRegime: "hybrid",
      results: [create(SearchHitSchema, {
        id: "symbol:demoteProvider",
        title: "demoteProvider",
        text: "Demotes a provider after a bounded failure threshold.",
        score: 0.912,
        path: "api/internal/retrieval/hybrid.go",
        startLine: 87,
        endLine: 102,
        analyzer: "go-code-graph",
        factKind: "symbol",
        proofStatus: "proven",
        sourceHash: "sha256:trusted",
        generation: "gen-active",
        retrievalRegime: "hybrid",
        retrievalExplanation: "lexical and semantic agreement",
        rankFactors: [create(SearchRankFactorSchema, { name: "rrf", leg: "fusion", value: 0.73 })],
        edgeExpansions: [create(SearchExpansionSchema, { id: "caller:search", title: "Search", path: "api/internal/retrieval/hybrid.go" })],
      })],
    }));
    renderWithProviders(<FactsWorkbench />);

    await user.click(screen.getByTestId(selectors.facts.searchButton));

    expect(await screen.findAllByText("demoteProvider")).toHaveLength(2);
    expect(screen.getByTestId(selectors.facts.searchResults)).toHaveTextContent("0.912");
    expect(screen.getByTestId(selectors.facts.provenancePanel)).toHaveTextContent("sha256:trusted");
    expect(screen.getByTestId(selectors.facts.provenancePanel)).toHaveTextContent("lexical and semantic agreement");
    expect(screen.getByTestId(selectors.facts.provenancePanel)).toHaveTextContent("Search");
    expect(vi.mocked(searchCodeFacts)).toHaveBeenCalledWith(expect.objectContaining({ query: "provider demotion", scope: "" }));
  });

  it("applies scoped search filters and renders an honest empty result", async () => {
    const user = userEvent.setup();
    vi.mocked(getIndexStatus).mockResolvedValueOnce(create(IndexStatusSchema, {
      state: "degraded",
      degradedStages: ["semantic"],
    }));
    vi.mocked(searchCodeFacts).mockResolvedValueOnce(create(SearchResponseSchema, {
      retrievalRegime: "lexical",
      degradedStages: ["graph"],
    }));
    renderWithProviders(<FactsWorkbench />);

    await user.selectOptions(screen.getByLabelText("Scope"), "api");
    await user.selectOptions(screen.getByLabelText("Role"), "definition");
    await user.selectOptions(screen.getByLabelText("Fact family"), String(FactFamily.SYMBOLS));
    await user.click(screen.getByTestId(selectors.facts.searchButton));

    expect(await screen.findByText("No indexed evidence matched. Try fewer filters or inspect index readiness.")).toBeInTheDocument();
    expect(screen.getByTestId(selectors.facts.degradedBanner)).toHaveTextContent("semantic, graph");
    expect(screen.getByTestId(selectors.facts.indexStatus)).toHaveTextContent("Index not promoted");
    expect(vi.mocked(searchCodeFacts)).toHaveBeenCalledWith(expect.objectContaining({
      scope: "api",
      roles: ["definition"],
      families: [FactFamily.SYMBOLS],
    }));
  });

  it("shows job progress and transparent fallback provenance", async () => {
    const user = userEvent.setup();
    vi.mocked(getIndexStatus).mockResolvedValueOnce(create(IndexStatusSchema, {
      activeGeneration: "gen-active",
      state: "building",
      sourceFiles: 1n,
      storageBytes: 1536n,
      lastReconcileAtUnix: BigInt(Math.floor(Date.now() / 1000)),
      activeJobs: [create(IndexJobSchema, { id: "job-1", kind: "reindex", processed: 5n, total: 10n })],
    }));
    vi.mocked(searchCodeFacts).mockResolvedValueOnce(create(SearchResponseSchema, {
      results: [create(SearchHitSchema, { id: "fallback", title: "Fallback evidence", score: 0.5 })],
    }));
    renderWithProviders(<FactsWorkbench />);

    await user.click(await screen.findByText("Corpus readiness"));
    expect(screen.getByRole("progressbar", { name: "reindex progress" })).toHaveAttribute("aria-valuenow", "50");
    expect(screen.getByText(/1 files · 1\.5 KiB/)).toBeInTheDocument();
    expect(screen.getByText("Just now")).toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.facts.searchButton));
    expect(await screen.findAllByText("Fallback evidence")).toHaveLength(2);
    expect(screen.getByTestId(selectors.facts.provenancePanel)).toHaveTextContent("source unavailable");
    expect(screen.getByTestId(selectors.facts.provenancePanel)).toHaveTextContent("did not expose component scores");
    expect(screen.getByTestId(selectors.facts.provenancePanel)).toHaveTextContent("No bounded graph neighbors");
    await user.click(screen.getByRole("button", { name: "Close evidence detail" }));
    expect(screen.getByTestId(selectors.facts.provenancePanel)).toHaveTextContent("Select a result");
  });

  it("surfaces search failures and prevents empty submissions", async () => {
    const user = userEvent.setup();
    vi.mocked(searchCodeFacts).mockRejectedValueOnce(new Error("search deadline exceeded"));
    renderWithProviders(<FactsWorkbench />);

    const searchInput = screen.getByTestId(selectors.facts.searchInput);
    await user.clear(searchInput);
    expect(screen.getByTestId(selectors.facts.searchButton)).toBeDisabled();
    await user.type(searchInput, "provider");
    await user.click(screen.getByTestId(selectors.facts.searchButton));
    expect(await screen.findByRole("alert")).toHaveTextContent("search deadline exceeded");
  });

  it("analyzes the selected target and renders report evidence", async () => {
    const user = userEvent.setup();
    vi.mocked(describeCodeFacts).mockResolvedValueOnce(makeReport());
    renderWithProviders(<FactsWorkbench />);

    await user.click(screen.getByTestId(selectors.facts.familyToggle({ family: String(FactFamily.CALLS) })));
    await user.click(screen.getByTestId(selectors.facts.analyzeButton));

    await waitFor(() => expect(vi.mocked(describeCodeFacts)).toHaveBeenCalledTimes(1));
    expect(vi.mocked(describeCodeFacts)).toHaveBeenCalledWith(
      expect.objectContaining({
        include: expect.not.arrayContaining([FactFamily.CALLS]),
        useCache: true,
      }),
    );

    expect(screen.getByTestId(selectors.facts.summary)).toHaveTextContent("1");
    expect(screen.getByTestId(selectors.facts.targetContext)).toHaveTextContent("code-facts");
    expect(screen.getByTestId(selectors.facts.cachePanel)).toHaveTextContent("describe");
    expect(screen.getByTestId(selectors.facts.surfacesTable)).toHaveTextContent("api");
    expect(screen.getByTestId(selectors.facts.parseUnitsTable)).toHaveTextContent("go-api");
    expect(screen.getByTestId(selectors.facts.factsTable)).toHaveTextContent("WriteProto");
    expect(screen.getByTestId(selectors.facts.evidenceTable)).toHaveTextContent("handlers/health.go");
    expect(screen.getByTestId(selectors.facts.warningsPanel)).toHaveTextContent("provider unavailable");
  });

  it("sends path targets and cache preference from the controls", async () => {
    const user = userEvent.setup();
    vi.mocked(describeCodeFacts).mockResolvedValueOnce(makeReport());
    renderWithProviders(<FactsWorkbench />);

    await user.selectOptions(screen.getByTestId(selectors.facts.targetKind), "path");
    await user.clear(screen.getByTestId(selectors.facts.targetInput));
    await user.type(screen.getByTestId(selectors.facts.targetInput), "/tmp/example");
    await user.click(screen.getByTestId(selectors.facts.cacheToggle));
    await user.click(screen.getByTestId(selectors.facts.analyzeButton));

    await waitFor(() => expect(vi.mocked(describeCodeFacts)).toHaveBeenCalledTimes(1));
    expect(vi.mocked(describeCodeFacts)).toHaveBeenCalledWith(
      expect.objectContaining({
        target: expect.objectContaining({
          kind: TargetKind.PATH,
          path: "/tmp/example",
        }),
        useCache: false,
      }),
    );
  });

  it("uses the surfaces family as a safe fallback when an operator deselects every family", async () => {
    const user = userEvent.setup();
    vi.mocked(describeCodeFacts).mockResolvedValueOnce(makeReport());
    renderWithProviders(<FactsWorkbench />);

    for (const family of [
      FactFamily.SURFACES,
      FactFamily.PARSE_UNITS,
      FactFamily.IMPORTS,
      FactFamily.SYMBOLS,
      FactFamily.REFERENCES,
      FactFamily.CALLS,
      FactFamily.PROTO_ADOPTION,
      FactFamily.ENDPOINT_PROOFS,
    ]) {
      await user.click(screen.getByTestId(selectors.facts.familyToggle({ family: String(family) })));
    }
    await user.click(screen.getByTestId(selectors.facts.analyzeButton));

    await waitFor(() => expect(vi.mocked(describeCodeFacts)).toHaveBeenCalledTimes(1));
    expect(vi.mocked(describeCodeFacts)).toHaveBeenCalledWith(
      expect.objectContaining({ include: [FactFamily.SURFACES] }),
    );
  });

  it.each([
    ["module", "/repo/module", TargetKind.MODULE],
    ["project", "/repo/project", TargetKind.PROJECT],
  ] as const)("builds a %s target from the operator controls", async (mode, value, kind) => {
    const user = userEvent.setup();
    vi.mocked(describeCodeFacts).mockResolvedValueOnce(makeReport());
    renderWithProviders(<FactsWorkbench />);

    await user.selectOptions(screen.getByTestId(selectors.facts.targetKind), mode);
    await user.clear(screen.getByTestId(selectors.facts.targetInput));
    await user.type(screen.getByTestId(selectors.facts.targetInput), value);
    await user.click(screen.getByTestId(selectors.facts.analyzeButton));

    await waitFor(() => expect(vi.mocked(describeCodeFacts)).toHaveBeenCalledTimes(1));
    expect(vi.mocked(describeCodeFacts)).toHaveBeenCalledWith(
      expect.objectContaining({ target: expect.objectContaining({ kind, path: value }) }),
    );
  });

  it("renders a complete but empty report without inventing evidence", async () => {
    const user = userEvent.setup();
    vi.mocked(describeCodeFacts).mockResolvedValueOnce(create(CodeFactsReportSchema, {}));
    renderWithProviders(<FactsWorkbench />);

    await user.click(screen.getByTestId(selectors.facts.analyzeButton));

    expect(await screen.findByTestId(selectors.facts.summary)).toHaveTextContent("0");
    expect(screen.getByTestId(selectors.facts.cachePanel)).toHaveTextContent("miss");
    expect(screen.getByTestId(selectors.facts.surfacesTable)).toHaveTextContent("facts.noSurfaces");
    expect(screen.getByTestId(selectors.facts.parseUnitsTable)).toHaveTextContent("facts.noParseUnits");
    expect(screen.getByTestId(selectors.facts.factsTable)).toHaveTextContent("facts.noFacts");
    expect(screen.getByTestId(selectors.facts.evidenceTable)).toHaveTextContent("facts.noEvidence");
    expect(screen.getByTestId(selectors.facts.warningsPanel)).toHaveTextContent("facts.noWarnings");
  });

  it("renders API errors in the console", async () => {
    const user = userEvent.setup();
    vi.mocked(describeCodeFacts).mockRejectedValueOnce(new Error("provider unreachable"));
    renderWithProviders(<FactsWorkbench />);

    await user.click(screen.getByTestId(selectors.facts.analyzeButton));

    const error = await screen.findByTestId(selectors.facts.error);
    expect(error).toHaveTextContent("provider unreachable");
  });
});

function makeReport() {
  const evidence = create(EvidenceSchema, {
    status: EvidenceStatus.PROVEN,
    analyzer: "go-code-graph",
    symbol: "WriteProto",
    message: "typed proto response writer",
    range: create(SourceRangeSchema, {
      file: "api/handlers/health.go",
      startLine: 17,
      startColumn: 3,
      endLine: 17,
      endColumn: 24,
    }),
  });

  return create(CodeFactsReportSchema, {
    target: create(TargetContextSchema, {
      resolvedKind: TargetKind.SCENARIO,
      rootPath: "/repo/scenarios/code-facts",
      scenario: "code-facts",
      scenarioAware: true,
    }),
    surfaces: [
      create(SurfaceSchema, {
        id: "api",
        kind: SurfaceKind.API,
        path: "api",
        status: SurfaceStatus.KNOWN,
        evidence: [evidence],
      }),
    ],
    parseUnits: [
      create(ParseUnitSchema, {
        id: "go-api",
        language: "go",
        rootPath: "api",
        status: EvidenceStatus.PROVEN,
        evidence: [evidence],
      }),
    ],
    facts: [
      create(GenericFactSchema, {
        id: "endpoint-proof:health",
        family: FactFamily.ENDPOINT_PROOFS,
        kind: "rest_exception",
        subject: "health",
        evidence: [evidence],
        attributes: { helper: "WriteProto" },
      }),
    ],
    evidence: [evidence],
    warnings: [
      create(WarningSchema, {
        code: "provider_unavailable",
        message: "provider unavailable",
        status: EvidenceStatus.UNKNOWN,
      }),
    ],
    cache: create(CacheMetadataSchema, {
      cacheKey: "cache-key",
      hit: true,
      state: "hit",
      reason: "fresh",
      scope: "describe",
      sourceHash: "src",
      configHash: "cfg",
      providerVersion: "go-code-graph",
      hitCount: 2n,
    }),
  });
}
