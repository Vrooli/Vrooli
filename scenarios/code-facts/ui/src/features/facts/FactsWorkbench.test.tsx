import { create } from "@bufbuild/protobuf";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { FactsWorkbench } from "./FactsWorkbench";
import {
  CacheMetadataSchema,
  CodeFactsReportSchema,
  EvidenceSchema,
  EvidenceStatus,
  FactFamily,
  GenericFactSchema,
  ParseUnitSchema,
  SourceRangeSchema,
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
  };
});

const { describeCodeFacts } = await import("../../api/facts");

describe("FactsWorkbench", () => {
  beforeEach(() => {
    vi.mocked(describeCodeFacts).mockReset();
  });

  it("renders an empty operator console before analysis runs", () => {
    renderWithProviders(<FactsWorkbench />);

    expect(screen.getByTestId(selectors.facts.workbench)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.facts.empty)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.facts.targetInput)).toHaveValue("code-facts");
    expect(screen.getByTestId(selectors.facts.familyControls)).toBeInTheDocument();
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
