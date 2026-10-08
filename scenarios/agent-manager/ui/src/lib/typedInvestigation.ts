export interface TypedInvestigationRequest {
  schemaVersion: "investigation-request/v1";
  requestKey: string;
  callerAuthority: "service";
  subject: {
    owner: "agent-manager";
    kind: "run-set";
    ref: string;
    revision: "current";
    runIds: string[];
  };
  question: string;
  evidencePolicy: {
    mode: "bounded_current";
    requiredPlanes: string[];
    optionalPlanes: string[];
    maxEvents: number;
    maxEvidenceBytes: number;
    maxReconciliations: number;
  };
  budget: {
    maxDelegatedRuns: 1;
    maxTurns: number;
    wallSeconds: 600;
    maxChargeMicroUsd: 1000000;
  };
  recommendationPolicy: {
    allowedKinds: ["observe", "recommend_action"];
    allowSubjectMutation: false;
  };
  provenance: {
    kind: "agent-manager-ui";
  };
}

export function buildTypedInvestigationRequest(
  runIds: string[],
  question: string,
  depth: "quick" | "standard" | "deep",
  requestKey: string,
): TypedInvestigationRequest {
  const subjectRunIds = Array.from(new Set(runIds.map((value) => value.trim()).filter(Boolean))).sort();
  if (subjectRunIds.length === 0) {
    throw new Error("At least one subject run is required");
  }
  const boundedQuestion = question.trim() || "Provide a bounded diagnosis of the selected agent runs, including supported findings and any unproven predicates.";
  const maxTurns = depth === "quick" ? 3 : depth === "deep" ? 16 : 8;
  return {
    schemaVersion: "investigation-request/v1",
    requestKey,
    callerAuthority: "service",
    subject: {
      owner: "agent-manager",
      kind: "run-set",
      ref: `run-set:${subjectRunIds.join(",")}`,
      revision: "current",
      runIds: subjectRunIds,
    },
    question: boundedQuestion,
    evidencePolicy: {
      mode: "bounded_current",
      requiredPlanes: ["run_state", "events", "invocations"],
      optionalPlanes: ["receipts", "diff"],
      maxEvents: 512,
      maxEvidenceBytes: 262144,
      maxReconciliations: 1,
    },
    budget: {
      maxDelegatedRuns: 1,
      maxTurns,
      wallSeconds: 600,
      maxChargeMicroUsd: 1000000,
    },
    recommendationPolicy: {
      allowedKinds: ["observe", "recommend_action"],
      allowSubjectMutation: false,
    },
    provenance: {
      kind: "agent-manager-ui",
    },
  };
}
