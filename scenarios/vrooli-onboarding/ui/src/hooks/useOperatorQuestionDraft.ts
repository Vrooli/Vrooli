import { useEffect, useMemo, useRef, useSyncExternalStore } from "react";
import { useQuery } from "@tanstack/react-query";
import { createFormStore, type FormStore } from "@vrooli/react-component-library/FormStore/1";
import { fetchOperatorState } from "../api/operatorstate";
import { discardDraft, fetchDraft, saveDraft } from "../api/session";
import { operatorStateRevision } from "../api/operatorStateRevision";
import { decodeOperatorQuestionDraft, encodeOperatorQuestionDraft, operatorQuestionDefaultValue, type DraftableQuestion } from "../lib/operatorQuestionDraft";

const draftActor = "onboarding-questionnaire";
const draftStep = "credentials";
const declinedField = "__vrooli_question_declined__";

type QuestionValues = Record<string, unknown>;

function initialValues(questions: DraftableQuestion[], choices?: Record<string, string>): QuestionValues {
  const restored = decodeOperatorQuestionDraft(questions, choices);
  const values: QuestionValues = {};
  for (const question of questions) {
    values[question.id] = restored.values[question.id] ?? operatorQuestionDefaultValue(question);
  }
  values[declinedField] = JSON.stringify(restored.declined);
  return values;
}

function declinedValues(values: QuestionValues): Record<string, boolean> {
  try {
    const parsed: unknown = JSON.parse(String(values[declinedField] ?? "{}"));
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return {};
    return Object.fromEntries(Object.entries(parsed).filter((entry): entry is [string, boolean] => typeof entry[1] === "boolean"));
  } catch {
    return {};
  }
}

/**
 * Persists safe operator answers through SessionService's target/actor draft
 * authority. Secret-kind inputs never enter the draft serializer.
 */
export function useOperatorQuestionDraft(target: string, questions: DraftableQuestion[]) {
  const questionSignature = JSON.stringify(questions.map(({ id, kind, defaultValue }) => [id, kind, defaultValue]));
  const draftQuery = useQuery({
    queryKey: ["operator-question-draft", target],
    queryFn: () => fetchDraft(target, draftActor),
    staleTime: 0,
    retry: false,
    refetchOnWindowFocus: false,
    enabled: questions.length > 0,
  });
  const draftRevision = draftQuery.data?.revision ?? "";
  const draftChoices = draftQuery.data?.choices ?? {};
  const choicesSignature = JSON.stringify(draftChoices);
  const acknowledgedRevision = useRef<Record<string, string>>({});
  const acknowledgedChoices = useRef<Record<string, string>>({});
  const writeQueue = useRef<Promise<void>>(Promise.resolve());
  const persistRef = useRef<(values: QuestionValues) => Promise<void>>(async () => undefined);

  persistRef.current = async (values) => {
    const choices = encodeOperatorQuestionDraft(questions, values, declinedValues(values));
    const serializedChoices = JSON.stringify(choices);
    const targetKey = target;
    if (acknowledgedChoices.current[targetKey] === serializedChoices) return;

    const write = async () => {
      const currentDraft = await fetchDraft(targetKey, draftActor);
      const expectedDraftRevision = acknowledgedRevision.current[targetKey] ?? draftRevision;
      if (currentDraft.revision && currentDraft.revision !== expectedDraftRevision && JSON.stringify(currentDraft.choices ?? {}) !== JSON.stringify(choices)) {
        throw new Error("These setup answers changed in another session. Reload the questions before saving again.");
      }

      const state = await fetchOperatorState(targetKey);
      const documentRevision = operatorStateRevision(state);
      const saved = await saveDraft({
        target: targetKey,
        actor: draftActor,
        baseRevision: currentDraft.baseRevision || documentRevision,
        expectedRevision: documentRevision,
        stepId: draftStep,
        choices,
      });
      acknowledgedRevision.current[targetKey] = saved.revision || expectedDraftRevision;
      acknowledgedChoices.current[targetKey] = serializedChoices;
    };

    const nextWrite = writeQueue.current.catch(() => undefined).then(write);
    writeQueue.current = nextWrite;
    return nextWrite;
  };

  const store = useMemo<FormStore<QuestionValues>>(() => createFormStore<QuestionValues>({
    initialValues: initialValues(questions, draftChoices),
    autosaveDelay: 400,
    autosave: async (values) => {
      await persistRef.current(values);
      return undefined;
    },
  }), [target, questionSignature, draftRevision, choicesSignature]);

  const snapshot = useSyncExternalStore(
    (notify) => store.subscribe(notify),
    () => store.get(),
    () => store.get(),
  );

  useEffect(() => () => {
    const dirty = Object.values(store.get().fields).some((field) => field.dirty);
    if (dirty) void store.save();
  }, [store]);

  const setDeclined = (requestID: string, value: boolean) => {
    const values = store.getValues();
    const declined = declinedValues(values);
    store.setValue(declinedField, JSON.stringify({ ...declined, [requestID]: value }));
  };
  const clear = async () => {
    await store.save();
    await writeQueue.current.catch(() => undefined);
    await discardDraft(target, draftActor);
  };
  const currentValues = store.getValues();

  return {
    store,
    declined: declinedValues(currentValues),
    loading: questions.length > 0 && draftQuery.isLoading,
    loadError: Boolean(draftQuery.error),
    phase: snapshot.phase,
    saveError: snapshot.error,
    retry: () => store.save(),
    retryLoad: () => draftQuery.refetch(),
    clear,
    setDeclined,
  };
}
