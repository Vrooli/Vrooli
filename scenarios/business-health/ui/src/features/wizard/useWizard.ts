import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type {
  Answer,
  ScaffoldPreview,
  ScaffoldResult,
  SessionState,
} from "@vrooli/proto-types/business-health/v1/wizard/wizard_pb";

import { wizardClient } from "../../api/wizard";

export interface StartSessionInput {
  readonly scenario: string;
  readonly reset: boolean;
}

/**
 * Drive one contract-authoring interview: start (or resume) a session, submit
 * answers question-by-question, preview the deterministic scaffold, and apply
 * it. The live `SessionState` is held here (seeded by start/submit) so the page
 * can render the current question, progress, and saved answers without a
 * separate query. Preview/apply results are held too so they survive re-renders
 * but reset whenever a new session starts.
 */
export function useWizard() {
  const [state, setState] = useState<SessionState | null>(null);
  const [preview, setPreview] = useState<ScaffoldPreview | null>(null);
  const [result, setResult] = useState<ScaffoldResult | null>(null);

  const start = useMutation<SessionState, unknown, StartSessionInput>({
    mutationFn: ({ scenario, reset }) => wizardClient.startSession({ scenario, reset }),
    onSuccess: (next) => {
      setState(next);
      setPreview(null);
      setResult(null);
    },
  });

  const submit = useMutation<SessionState, unknown, Answer[]>({
    mutationFn: (answers) => {
      const sessionId = state?.sessionId ?? "";
      return wizardClient.submitAnswers({ sessionId, answers });
    },
    onSuccess: (next) => {
      setState(next);
      // Answers changed — any prior preview/result is stale.
      setPreview(null);
      setResult(null);
    },
  });

  const previewScaffold = useMutation<ScaffoldPreview, unknown>({
    mutationFn: () => {
      const sessionId = state?.sessionId ?? "";
      return wizardClient.previewScaffold({ sessionId });
    },
    onSuccess: (next) => setPreview(next),
  });

  const applyScaffold = useMutation<ScaffoldResult, unknown>({
    mutationFn: () => {
      const sessionId = state?.sessionId ?? "";
      return wizardClient.applyScaffold({ sessionId, apply: true });
    },
    onSuccess: (next) => setResult(next),
  });

  return { state, preview, result, start, submit, previewScaffold, applyScaffold };
}
