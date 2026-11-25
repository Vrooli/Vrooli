import { describe, expect, it } from "vitest";
import { create, fromJson, toJson } from "@bufbuild/protobuf";
import { screen } from "@testing-library/react";
import {
  CorrectionCandidate_StrengthBand,
  QueryPartEvidence_Source,
  QueryPartEvidence_Status,
  QueryPartEvidenceSchema,
  QueryResponseSchema,
  SearchHitReference_RetrievalSource,
  type QueryPartEvidence,
} from "@vrooli/proto-types/search-hub/v1/routing/routing_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { SearchResults } from "./SearchResults";

const FALLBACK_ID = "fallback-id";
const MATCHING_SNIPPET = "A matching document";
const DOWN_PROVIDER_NOTE = "temporarily unavailable";
const WEAK_RESULT_TITLE = "Weak result";

describe("SearchResults", () => {
  it("renders the unified ranking with a weak, located result", () => {
    const data = create(QueryResponseSchema, {
      reranked: true,
      ranked: [{
        providerId: "docs",
        providerGroup: "knowledge-observatory",
        id: FALLBACK_ID,
        title: "",
        snippet: MATCHING_SNIPPET,
        path: "",
        locations: ["one.md", "two.md", "three.md"],
        confidence: { weak: true, regime: "hybrid" },
      }],
    });

    renderWithProviders(<SearchResults data={data} />);

    expect(screen.getByTestId(selectors.search.results)).toBeInTheDocument();
    expect(screen.getByText(FALLBACK_ID)).toBeInTheDocument();
    expect(screen.getByText(MATCHING_SNIPPET)).toBeInTheDocument();
    expect(screen.getByText(strings.search.locations)).toBeInTheDocument();
  });

  it("renders degraded, empty, and all-weak provider groups honestly", () => {
    const data = create(QueryResponseSchema, {
      groups: [
        { providerId: "down-provider", degraded: true, note: DOWN_PROVIDER_NOTE },
        { providerId: "empty-provider", count: 0 },
        {
          providerId: "weak-provider",
          count: 1,
          hits: [{
            providerId: "weak-provider",
            id: "weak-id",
            title: WEAK_RESULT_TITLE,
            confidence: { weak: true },
          }],
        },
        {
          providerId: "partial-provider",
          count: 1,
          degraded: true,
          note: "semantic leg timed out",
          hits: [{ id: "partial-id", title: "Valid partial result", snippet: MATCHING_SNIPPET }],
        },
      ],
    });

    renderWithProviders(<SearchResults data={data} />);

    expect(screen.getByText(DOWN_PROVIDER_NOTE)).toBeInTheDocument();
    expect(screen.getByText("semantic leg timed out")).toBeInTheDocument();
    expect(screen.getByText("Valid partial result")).toBeInTheDocument();
    expect(screen.getByText(WEAK_RESULT_TITLE)).toBeInTheDocument();
    expect(screen.getAllByText(new RegExp(strings.search.groupEmpty)).length).toBeGreaterThan(0);
    expect(screen.getAllByText(new RegExp(strings.search.noConfidentMatch)).length).toBeGreaterThan(0);
  });

  it("keeps every part status visible while applying one four-hit evidence budget", () => {
    const hits = Array.from({ length: 5 }, (_, index) => ({
      providerId: "ui-health.surfaces",
      id: `surface-${index + 1}`,
      title: `Health surface ${index + 1}`,
      path: `scenarios/ui-health/surface-${index + 1}.md`,
    }));
    const parts: QueryPartEvidence[] = [0, 1, 2, 3].map((index) => create(QueryPartEvidenceSchema, {
      partId: `covered-${index + 1}`,
      label: `Covered request part ${index + 1}`,
      status: QueryPartEvidence_Status.CANDIDATES_RETURNED,
      evidence: [{ providerId: "ui-health.surfaces", groupIndex: 0, hitIndex: index, hitId: hits[index]?.id ?? "" }],
    }));
    parts.push(
      create(QueryPartEvidenceSchema, {
        partId: "missing",
        label: "Later missing requirement",
        status: QueryPartEvidence_Status.NOT_ROUTED,
        reason: "ambiguous_provider_evidence",
        evidence: [],
      }),
      create(QueryPartEvidenceSchema, {
        partId: "over-budget",
        label: "Later returned requirement",
        status: QueryPartEvidence_Status.CANDIDATES_RETURNED,
        evidence: [{ providerId: "ui-health.surfaces", groupIndex: 0, hitIndex: 4, hitId: hits[4]?.id ?? "" }],
      }),
    );
    const data = create(QueryResponseSchema, {
      groups: [{ providerId: "ui-health.surfaces", count: 5, hits }],
      queryParts: parts,
    });

    renderWithProviders(<SearchResults data={data} />);

    expect(screen.getByText(strings.search.taskHeading)).toBeInTheDocument();
    expect(screen.getByText("Later missing requirement")).toBeInTheDocument();
    expect(screen.getByText(/search\.partNotRouted/)).toBeInTheDocument();
    expect(screen.getByText("Later returned requirement")).toBeInTheDocument();
    expect(screen.getByText(strings.search.evidenceOmitted).getAttribute("data-omitted-count")).toBe("1");
    expect(screen.queryByText("Health surface 5")).not.toBeInTheDocument();
  });

  it("shows all task-part hits and counts globally omitted identities once", () => {
    const hits = Array.from({ length: 6 }, (_, index) => ({
      providerId: "catalog.docs",
      id: `candidate-${index + 1}`,
      title: `Candidate evidence ${index + 1}`,
      path: `scenarios/catalog/candidate-${index + 1}.md`,
    }));
    const ref = (index: number) => ({
      providerId: "catalog.docs", groupIndex: 0, hitIndex: index, hitId: hits[index]?.id ?? "",
    });
    const data = create(QueryResponseSchema, {
      groups: [{ providerId: "catalog.docs", hits }],
      queryParts: [
        { partId: "first", label: "Compare initial candidates", status: QueryPartEvidence_Status.CANDIDATES_RETURNED, evidence: [ref(0), ref(1), ref(2)] },
        { partId: "budget", label: "Inspect next candidates", status: QueryPartEvidence_Status.CANDIDATES_RETURNED, evidence: [ref(3), ref(4), ref(4)] },
        { partId: "overlap", label: "Check overlapping evidence", status: QueryPartEvidence_Status.CANDIDATES_RETURNED, evidence: [ref(4), ref(5)] },
      ],
    });

    renderWithProviders(<SearchResults data={data} />);

    for (const title of ["Candidate evidence 1", "Candidate evidence 2", "Candidate evidence 3", "Candidate evidence 4"]) {
      expect(screen.getByText(title)).toBeInTheDocument();
    }
    expect(screen.queryByText("Candidate evidence 5")).not.toBeInTheDocument();
    expect(screen.queryByText("Candidate evidence 6")).not.toBeInTheDocument();
    const perPartOmissions = screen.getAllByText(strings.search.evidenceOmitted);
    expect(perPartOmissions.map((element) => element.getAttribute("data-omitted-count"))).toEqual(["1", "2"]);
    expect(screen.getByText(strings.search.omittedHits).getAttribute("data-omitted-count")).toBe("2");
  });

  it("prioritizes request-part hits over optional correction evidence under the shared budget", () => {
    const hits = Array.from({ length: 5 }, (_, index) => ({
      providerId: "docs",
      id: `task-hit-${index + 1}`,
      title: `Task evidence ${index + 1}`,
      path: `docs/task-${index + 1}.md`,
    }));
    const ref = (index: number) => ({ providerId: "docs", groupIndex: 0, hitIndex: index, hitId: hits[index]?.id ?? "" });
    const data = create(QueryResponseSchema, {
      groups: [{ providerId: "docs", hits }],
      queryParts: [0, 1, 2, 3].map((index) => ({
        partId: `part-${index + 1}`,
        label: `Task part ${index + 1}`,
        status: QueryPartEvidence_Status.CANDIDATES_RETURNED,
        evidence: [ref(index)],
      })),
      correctionCandidates: [{
        submittedValue: "TaskEvidenc",
        candidateValue: "TaskEvidence",
        strength: CorrectionCandidate_StrengthBand.POSSIBLE,
        evidence: ref(4),
      }],
    });

    renderWithProviders(<SearchResults data={data} />);

    for (const title of ["Task evidence 1", "Task evidence 2", "Task evidence 3", "Task evidence 4"]) {
      expect(screen.getByText(title)).toBeInTheDocument();
    }
    expect(screen.queryByText("Task evidence 5")).not.toBeInTheDocument();
    expect(screen.queryByText(/TaskEvidenc → TaskEvidence/)).not.toBeInTheDocument();
    expect(screen.getByText((content) => content.includes(strings.search.omittedCorrections))).toBeInTheDocument();
  });

  it("marks candidate parts whose supporting-hit reference is absent or invalid", () => {
    const data = create(QueryResponseSchema, {
      groups: [{ providerId: "ui-health.surfaces", hits: [{ id: "real-hit", title: "A returned surface" }] }],
      queryParts: [
        {
          partId: "empty-reference",
          label: "Candidate part with no reference",
          status: QueryPartEvidence_Status.CANDIDATES_RETURNED,
        },
        {
          partId: "invalid-reference",
          label: "Candidate part with stale identity",
          status: QueryPartEvidence_Status.CANDIDATES_RETURNED,
          evidence: [{ providerId: "ui-health.surfaces", groupIndex: 0, hitIndex: 0, hitId: "different-hit" }],
        },
      ],
    });

    renderWithProviders(<SearchResults data={data} />);

    expect(screen.getAllByText(strings.search.supportUnavailable)).toHaveLength(2);
    expect(screen.queryByText("A returned surface")).not.toBeInTheDocument();
  });

  it("shows query provenance for visible derived-segment evidence, including duplicates", () => {
    const hit = { providerId: "docs", id: "health-card", title: "HealthCard component", path: "ui/HealthCard.tsx" };
    const evidence = {
      providerId: "docs",
      groupIndex: 0,
      hitIndex: 0,
      hitId: "health-card",
      retrievalSources: [
        SearchHitReference_RetrievalSource.DERIVED_SEGMENT,
        SearchHitReference_RetrievalSource.USER_QUERY,
      ],
    };
    const data = create(QueryResponseSchema, {
      groups: [{ providerId: "docs", hits: [hit] }],
      queryParts: [
        {
          partId: "surface",
          label: "Find interface from derived segment",
          source: QueryPartEvidence_Source.DERIVED_SEGMENT,
          status: QueryPartEvidence_Status.CANDIDATES_RETURNED,
          evidence: [evidence],
        },
        {
          partId: "route",
          label: "Find route using same candidate",
          source: QueryPartEvidence_Source.DERIVED_SEGMENT,
          status: QueryPartEvidence_Status.CANDIDATES_RETURNED,
          evidence: [evidence],
        },
      ],
    });

    renderWithProviders(<SearchResults data={data} />);

    expect(screen.getByText(strings.search.alsoFoundForFullQuery)).toBeInTheDocument();
    expect(screen.getByText((content) => content.includes(strings.search.duplicateEvidence))).toBeInTheDocument();
    expect(screen.getByText((content) => content.includes(strings.search.duplicateAlsoFoundForFullQuery))).toBeInTheDocument();
  });

  it("selects an exact entity view from the submitted query", () => {
    const data = create(QueryResponseSchema, {
      reranked: true,
      ranked: [
        { providerId: "docs", id: "health-card", title: "HealthCard surface", path: "scenarios/ui-health/HealthCard.tsx" },
        { providerId: "docs", id: "other", title: "Other result" },
      ],
    });

    renderWithProviders(<SearchResults data={data} query="HealthCard surface" />);

    expect(screen.getByText(strings.search.directHeading)).toBeInTheDocument();
    expect(screen.getByText("HealthCard surface")).toBeInTheDocument();
    expect(screen.getByText("Other result")).toBeInTheDocument();
  });

  it("prioritizes an exact direct hit while keeping other unique hits under the shared cap", () => {
    const hits = [
      { providerId: "docs", id: "alpha", title: "Alpha result", path: "scenarios/docs/alpha.md" },
      { providerId: "docs", id: "beta", title: "Beta result", path: "scenarios/docs/beta.md" },
      { providerId: "docs", id: "target", title: "Exact target", path: "scenarios/docs/target.md" },
      { providerId: "docs", id: "gamma", title: "Gamma result", path: "scenarios/docs/gamma.md" },
      { providerId: "docs", id: "delta", title: "Delta result", path: "scenarios/docs/delta.md" },
      { providerId: "docs", id: "epsilon", title: "Epsilon result", path: "scenarios/docs/epsilon.md" },
      { providerId: "docs-copy", id: "alpha-copy", title: "Duplicate alpha", path: "scenarios/docs/alpha.md#L20" },
    ];
    const data = create(QueryResponseSchema, {
      reranked: true,
      ranked: hits,
      groups: [{ providerId: "docs", count: 7, hits }],
    });

    renderWithProviders(<SearchResults data={data} query="Exact target" />);

    expect(screen.getByText(strings.search.directHeading)).toBeInTheDocument();
    expect(screen.getByText("Exact target")).toBeInTheDocument();
    expect(screen.getByText("Alpha result")).toBeInTheDocument();
    expect(screen.getByText("Gamma result")).toBeInTheDocument();
    expect(screen.queryByText("Duplicate alpha")).not.toBeInTheDocument();
    expect(screen.queryByText("Delta result")).not.toBeInTheDocument();
    expect(screen.getByText(strings.search.omittedHits).getAttribute("data-omitted-count")).toBe("2");
  });

  it("deduplicates one hit shared by corrections and multiple task parts", () => {
    const hit = {
      providerId: "ui-health.surfaces",
      id: "health-card",
      title: "HealthCard surface",
      path: "scenarios/ui-health/HealthCard.tsx",
    };
    const reference = { providerId: hit.providerId, groupIndex: 0, hitIndex: 0, hitId: hit.id };
    const data = create(QueryResponseSchema, {
      groups: [{ providerId: hit.providerId, count: 1, hits: [hit] }],
      correctionCandidates: [{
        submittedValue: "HealthCar",
        candidateValue: "HealthCard",
        strength: 1,
        evidence: reference,
      }],
      queryParts: [
        { partId: "owner", label: "Find the owning surface", status: QueryPartEvidence_Status.CANDIDATES_RETURNED, evidence: [reference] },
        { partId: "file", label: "Read the matching file", status: QueryPartEvidence_Status.CANDIDATES_RETURNED, evidence: [reference] },
      ],
    });

    renderWithProviders(<SearchResults data={data} query="HealthCar and find/read" />);

    expect(screen.getByText(/HealthCar → HealthCard/)).toBeInTheDocument();
    expect(screen.getAllByText(/HealthCard surface/)).toHaveLength(1);
    expect(screen.getAllByText(strings.search.duplicateEvidence)).toHaveLength(2);
  });

  it("decodes additive task coverage from Connect ProtoJSON", () => {
    const data = fromJson(QueryResponseSchema, {
      groups: [],
      queryParts: [{
        partId: "owner",
        label: "Find the owning capability",
        status: "STATUS_NOT_ROUTED",
        reason: "ambiguous_provider_evidence",
        evidence: [],
      }],
    });

    renderWithProviders(<SearchResults data={data} />);

    expect(screen.getByText("Find the owning capability")).toBeInTheDocument();
    expect(screen.getByText(/search\.partNotRouted/)).toBeInTheDocument();
  });

  it("renders task evidence and correction provenance after generated ProtoJSON round-trip", () => {
    const hits = [
      { providerId: "docs", id: "health-card", title: "HealthCard component", path: "ui/HealthCard.tsx" },
      { providerId: "docs", id: "route-guide", title: "Route guide", path: "docs/routes.md" },
    ];
    const ref = (hitIndex: number, hitId: string) => ({
      providerId: "docs", groupIndex: 0, hitIndex, hitId,
    });
    const typed = create(QueryResponseSchema, {
      groups: [{ providerId: "docs", count: hits.length, hits }],
      correctionCandidates: [{
        submittedValue: "HealthCar",
        candidateValue: "HealthCard",
        providerId: "docs",
        strength: CorrectionCandidate_StrengthBand.POSSIBLE,
        evidence: ref(0, "health-card"),
      }],
      queryParts: [
        {
          partId: "surface",
          label: "Find the relevant interface",
          status: QueryPartEvidence_Status.CANDIDATES_RETURNED,
          evidence: [ref(0, "health-card")],
        },
        {
          partId: "route",
          label: "Find routing guidance",
          status: QueryPartEvidence_Status.CANDIDATES_RETURNED,
          evidence: [ref(1, "stale-id")],
        },
        {
          partId: "owner",
          label: "Identify the owning capability",
          status: QueryPartEvidence_Status.NOT_ROUTED,
          reason: "ambiguous_provider_evidence",
          evidence: [],
        },
      ],
    });
    // Exercise the generated ProtoJSON producer and consumer before the real
    // renderer, including enum strings, lowerCamel fields, and typed refs.
    const wire = toJson(QueryResponseSchema, typed);
    expect(wire).toMatchObject({
      correctionCandidates: [{ submittedValue: "HealthCar", candidateValue: "HealthCard" }],
      queryParts: [
        { status: "STATUS_CANDIDATES_RETURNED", evidence: [{ hitId: "health-card" }] },
        { status: "STATUS_CANDIDATES_RETURNED", evidence: [{ hitId: "stale-id" }] },
        { status: "STATUS_NOT_ROUTED", reason: "ambiguous_provider_evidence" },
      ],
    });
    const response = fromJson(QueryResponseSchema, wire);

    renderWithProviders(<SearchResults data={response} query="HealthCar route" />);

    expect(screen.getByText(strings.search.taskHeading)).toBeInTheDocument();
    expect(screen.getByText(/HealthCar → HealthCard/)).toBeInTheDocument();
    expect(screen.getByText(/search\.supportUnavailable/)).toBeInTheDocument();
    expect(screen.getByText("Identify the owning capability")).toBeInTheDocument();
    expect(screen.getByText(/search\.partNotRouted/)).toBeInTheDocument();
    expect(screen.getAllByText(/HealthCard component/)).toHaveLength(1);
    expect(screen.queryByText("Route guide")).not.toBeInTheDocument();
  });
});
