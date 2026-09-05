import type { FixResponse } from "@vrooli/proto-types/scenario-validation/v1/validation_pb";
import type {
  ReconciliationEvidenceRow,
  ScenarioSpecPage,
  StudioApplyResult,
  StudioPageDraft,
} from "../api/experience";

export function displayDepth(page: ScenarioSpecPage) {
  const spec = page.spec;
  if ((spec.states?.length ?? 0) > 1) {
    return "L3";
  }
  if ((spec.elements?.length ?? 0) > 0 && (spec.claims?.length ?? 0) > 0) {
    return "L2";
  }
  if ((spec.priorities?.length ?? 0) > 0) {
    return "L1";
  }
  return "L0";
}

export function stateDepth(page: ScenarioSpecPage, state: string) {
  return page.spec.states?.some((entry) => entry.id === state) ? displayDepth(page) : "-";
}

export function machineClaims(pages: ScenarioSpecPage[]) {
  return pages.flatMap((page) =>
    (page.spec.claims ?? [])
      .filter((claim) => claim.tier === "machine")
      .map((claim) => ({ page, claim })),
  );
}

export function claimEvidencePath(scenario: string, pageID: string) {
  return `/scenarios/${scenario}/pages/${pageID}/evidence`;
}

export function newestEvidence(rows: ReconciliationEvidenceRow[]) {
  return [...rows].sort((a, b) => Date.parse(b.checkedAt) - Date.parse(a.checkedAt))[0];
}

export function captureImageSource(captureRef: string) {
  if (/^(https?:|data:|blob:|\/)/.test(captureRef)) {
    return captureRef;
  }
  return "";
}

export function evidenceIsStale(row: ReconciliationEvidenceRow | undefined) {
  if (!row?.checkedAt) {
    return false;
  }
  const checked = Date.parse(row.checkedAt);
  return Number.isFinite(checked) && Date.now() - checked > 5 * 60_000;
}

export function formatEvidenceMeta(row: ReconciliationEvidenceRow) {
  const viewport = row.viewport
    ? `${row.viewport}${row.viewportWidth && row.viewportHeight ? ` ${row.viewportWidth}x${row.viewportHeight}` : ""}`
    : "";
  return [row.claimType, viewport, row.checkedAt, row.verdict].filter(Boolean).join(" · ");
}

export function formatAXNode(json: string) {
  const trimmed = json.trim();
  if (trimmed === "" || trimmed === "{}") {
    return "";
  }
  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2);
  } catch {
    return trimmed;
  }
}

export function firstMachineClaim(page: ScenarioSpecPage | undefined) {
  return page?.spec.claims?.find((claim) => claim.tier === "machine") ?? page?.spec.claims?.[0];
}

export function pageDraftFromSpec(page: ScenarioSpecPage | undefined, title: string, claimStatement: string): StudioPageDraft {
  const spec = page?.spec;
  const pageID = spec?.page.id || page?.document.id || "new-page";
  const baseClaim = firstMachineClaim(page);
  const claimID = baseClaim?.id || `${pageID}-draft-claim`;
  const statement = baseClaim?.statement || claimStatement;
  return {
    id: pageID,
    title,
    purpose: spec?.page.purpose ?? "",
    routes: spec?.page.routes ?? [],
    prdRefs: spec?.page.prd_refs ?? [],
    status: page?.document.status || "draft",
    priorities: (spec?.priorities ?? []).map((priority) => ({
      statement: priority.statement,
      notes: priority.notes ?? "",
    })),
    states: (spec?.states ?? [{ id: "default", description: "" }]).map((state) => ({
      id: state.id,
      description: state.description ?? "",
    })),
    elements: (spec?.elements ?? []).map((element) => ({
      id: element.id,
      role: element.role ?? "",
      name: element.name ?? "",
      description: element.description ?? "",
    })),
    claims: [
      {
        id: claimID,
        type: baseClaim?.type ?? "custom",
        statement,
        tier: baseClaim?.tier ?? "machine",
        elements: baseClaim?.elements ?? [],
        states: baseClaim?.states ?? ["default"],
        viewports: [],
        locales: [],
        rationale: "",
      },
    ],
    bindings: [],
    sketchRegions: [],
  };
}

export function validationText(result: StudioApplyResult | undefined, fallback: string) {
  const findings = result?.validation?.report?.findings ?? [];
  if (findings.length === 0) {
    return fallback;
  }
  return findings.map((finding) => `${finding.severity}: ${finding.title}`).join("\n");
}

export function uniqueRuleIDs(response: FixResponse | undefined) {
  return Array.from(new Set(response?.candidates.map((candidate) => candidate.ruleId).filter(Boolean) ?? []));
}

export function fixPreviewText(response: FixResponse | undefined, emptyCopy: string) {
  if (!response) {
    return emptyCopy;
  }
  if (response.candidates.length === 0) {
    return response.messages.join("\n") || emptyCopy;
  }
  return response.candidates
    .map((candidate) => {
      const beforeLines = candidate.before ? candidate.before.split("\n").length : 0;
      const afterLines = candidate.after ? candidate.after.split("\n").length : 0;
      return `${candidate.ruleId} ${candidate.filePath}\n${candidate.description}\n-${beforeLines} +${afterLines}`;
    })
    .join("\n\n");
}
