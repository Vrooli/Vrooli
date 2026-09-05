import { useState, useCallback, useRef, useEffect, useMemo } from "react";
import { backlogService } from "../services/backlog-service";
import type { QuestionAnswer } from "../components/backlog/question-renderers";
import type { CrossItemQuestion } from "../lib/command-post-utils";
import type { BacklogKind } from "../types";
import {
  decisionParentKey,
  decisionQuestionKey,
  getUnresolvedDecisionQuestions,
  normalizeDecisionIndex,
} from "../lib/decision-stream-queue";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface DecisionStreamResults {
  answeredCount: number;
  skippedCount: number;
  snoozedCount: number;
  unlockedItems: {
    kind: string;
    name: string;
    title: string;
  }[];
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

export function groupByParent(questions: CrossItemQuestion[]): Map<string, CrossItemQuestion[]> {
  const map = new Map<string, CrossItemQuestion[]>();
  for (const ciq of questions) {
    const key = `${ciq.parentKind}/${ciq.parentName}`;
    const list = map.get(key);
    if (list) list.push(ciq);
    else map.set(key, [ciq]);
  }
  return map;
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export interface UseDecisionStreamLogicArgs {
  questions: CrossItemQuestion[];
  onComplete: (results: DecisionStreamResults) => void;
  onBack: () => void;
  onSnoozeItem: (key: string) => void;
  navigatorOpenRef?: React.RefObject<boolean>;
  toggleNavigator?: () => void;
  onQueueComplete?: () => void;
  currentQuestionId?: string | null;
  onCurrentQuestionChange?: (id: string | null) => void;
}

export function useDecisionStreamLogic({
  questions,
  onComplete,
  onBack,
  onSnoozeItem,
  navigatorOpenRef,
  toggleNavigator,
  onQueueComplete,
  currentQuestionId,
  onCurrentQuestionChange,
}: UseDecisionStreamLogicArgs) {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [localAnswers, setLocalAnswers] = useState<Map<string, QuestionAnswer>>(() => new Map());
  const [answeredQuestionKeys, setAnsweredQuestionKeys] = useState<Set<string>>(() => new Set());
  const [skippedIds, setSkippedIds] = useState<Set<string>>(() => new Set());
  const [deletedIds] = useState<Set<string>>(() => new Set());
  const [snoozedItemKeys, setSnoozedItemKeys] = useState<Set<string>>(() => new Set());
  const [savingId, setSavingId] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [phase, setPhase] = useState<"answering" | "completing" | "complete">("answering");
  const [completionResults, setCompletionResults] = useState<DecisionStreamResults | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const activeQuestions = getUnresolvedDecisionQuestions({
    questions,
    answeredKeys: answeredQuestionKeys,
    deletedIds,
    snoozedParentKeys: snoozedItemKeys,
  });
  const total = activeQuestions.length;
  const requestedIndex = currentQuestionId ? activeQuestions.findIndex((question) => question.question.id === currentQuestionId) : -1;
  const safeIndex = normalizeDecisionIndex(requestedIndex >= 0 ? requestedIndex : currentIndex, total);
  const current = activeQuestions[safeIndex] as CrossItemQuestion | undefined;
  const answer = current ? localAnswers.get(current.question.id) : undefined;

  useEffect(() => {
    const normalized = normalizeDecisionIndex(currentIndex, total);
    if (normalized !== currentIndex) setCurrentIndex(normalized);
  }, [currentIndex, total]);

  useEffect(() => {
    if (current && current.question.id !== currentQuestionId) onCurrentQuestionChange?.(current.question.id);
    if (!current && currentQuestionId) onCurrentQuestionChange?.(null);
  }, [current, currentQuestionId, onCurrentQuestionChange]);

  // ---------------------------------------------------------------------------
  // Answer management
  // ---------------------------------------------------------------------------

  const updateAnswer = useCallback((questionId: string, patch: Partial<QuestionAnswer>) => {
    setLocalAnswers((prev) => {
      const next = new Map(prev);
      next.set(questionId, { ...prev.get(questionId), ...patch });
      return next;
    });
    setSaveError(null);
  }, []);

  // ---------------------------------------------------------------------------
  // Save
  // ---------------------------------------------------------------------------

  const saveAnswer = useCallback(async (
    ciq: CrossItemQuestion,
    a: QuestionAnswer | undefined,
  ): Promise<boolean> => {
    if (!a) return false;
    const q = ciq.question;
    setSavingId(q.id);
    setSaveError(null);
    try {
      if (q.source === "review" && (a.reviewStatus === "approved" || a.reviewStatus === "flagged")) {
        await backlogService.batchReview(ciq.parentKind, ciq.parentName, [{
          id: q.id,
          type: q.review_type ?? "target",
          module_id: q.module_id,
          review_status: a.reviewStatus,
          review_comment: a.reviewComment ?? "",
        }]);
      }
      setAnsweredQuestionKeys((prev) => new Set(prev).add(decisionQuestionKey(ciq)));
      return true;
    } catch {
      setSaveError("Save failed — will retry on next advance");
      return false;
    } finally {
      setSavingId(null);
    }
  }, []);

  // ---------------------------------------------------------------------------
  // ---------------------------------------------------------------------------
  // Completion
  // ---------------------------------------------------------------------------

  const handleCompletion = useCallback((answeredOverride?: Set<string>) => {
    setPhase("completing");
    const effectiveAnswered = answeredOverride ?? answeredQuestionKeys;

    const answeredCount = Array.from(effectiveAnswered).length;
    const validLocalAnswerCount = Array.from(localAnswers.values()).filter((a) => {
      return a.reviewStatus === "approved" || a.reviewStatus === "flagged";
    }).length;

    const unlockedItems: DecisionStreamResults["unlockedItems"] = [];

    const results = {
      answeredCount: Math.max(answeredCount, validLocalAnswerCount),
      skippedCount: skippedIds.size,
      snoozedCount: snoozedItemKeys.size,
      unlockedItems,
    };
    if (onQueueComplete) {
      onQueueComplete();
      return;
    }
    setCompletionResults(results);
    setPhase("complete");
  }, [answeredQuestionKeys, localAnswers, skippedIds, snoozedItemKeys, onQueueComplete]);

  // ---------------------------------------------------------------------------
  // Navigation
  // ---------------------------------------------------------------------------

  const isAllDone = useCallback(() => {
    return activeQuestions.every((ciq) => {
      if (skippedIds.has(ciq.question.id)) return true;
      const a = localAnswers.get(ciq.question.id);
      if (!a) return false;
      return a.reviewStatus === "approved" || a.reviewStatus === "flagged";
    });
  }, [activeQuestions, skippedIds, localAnswers]);

  const advance = useCallback(async () => {
    if (!current) return;
    const a = localAnswers.get(current.question.id);
    let nextAnswered = answeredQuestionKeys;
    if (a) {
      const saved = await saveAnswer(current, a);
      if (!saved) return;
      nextAnswered = new Set(answeredQuestionKeys).add(decisionQuestionKey(current));
    }
    const remainingAfterAdvance = activeQuestions.filter((ciq) => !nextAnswered.has(decisionQuestionKey(ciq)));
    if (remainingAfterAdvance.length === 0 || isAllDone()) {
      handleCompletion(nextAnswered);
      return;
    }
    if (!a && safeIndex < total - 1) {
      setCurrentIndex(safeIndex + 1);
    }
  }, [current, safeIndex, total, localAnswers, answeredQuestionKeys, saveAnswer, activeQuestions, isAllDone, handleCompletion]);

  const goBack = useCallback(() => {
    if (safeIndex > 0) {
      setCurrentIndex(safeIndex - 1);
      setSaveError(null);
    }
  }, [safeIndex]);

  const skip = useCallback(() => {
    if (!current) return;
    const newSkipped = new Set(skippedIds);
    newSkipped.add(current.question.id);
    setSkippedIds(newSkipped);
    if (safeIndex < total - 1) {
      setCurrentIndex(safeIndex + 1);
    }
    const allDone = activeQuestions.every((ciq) => {
      if (newSkipped.has(ciq.question.id)) return true;
      const a = localAnswers.get(ciq.question.id);
      if (!a) return false;
      return a.reviewStatus === "approved" || a.reviewStatus === "flagged";
    });
    if (allDone) {
      void handleCompletion();
    }
  }, [current, safeIndex, total, skippedIds, activeQuestions, localAnswers, handleCompletion]);

  const snoozeParent = useCallback(() => {
    if (!current) return;
    const key = decisionParentKey(current.parentKind, current.parentName);
    const newSnoozed = new Set(snoozedItemKeys);
    newSnoozed.add(key);
    setSnoozedItemKeys(newSnoozed);
    onSnoozeItem(key);
  }, [current, snoozedItemKeys, onSnoozeItem]);

  // ---------------------------------------------------------------------------
  // Navigator helpers
  // ---------------------------------------------------------------------------

  const parentGroups = useMemo(() => groupByParent(activeQuestions), [activeQuestions]);

  const jumpToParent = useCallback((parentKey: string) => {
    const idx = activeQuestions.findIndex(
      (ciq) => `${ciq.parentKind}/${ciq.parentName}` === parentKey,
    );
    if (idx >= 0) setCurrentIndex(idx);
  }, [activeQuestions]);

  const snoozeSpecificParent = useCallback((kind: BacklogKind, name: string) => {
    const key = decisionParentKey(kind, name);
    const newSnoozed = new Set(snoozedItemKeys);
    newSnoozed.add(key);
    setSnoozedItemKeys(newSnoozed);
    onSnoozeItem(key);
  }, [snoozedItemKeys, onSnoozeItem]);

  // ---------------------------------------------------------------------------
  // Keyboard shortcuts
  // ---------------------------------------------------------------------------

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      const tag = (e.target as HTMLElement)?.tagName;
      if (tag === "TEXTAREA" || tag === "INPUT") return;
      if (phase !== "answering" || !current) return;

      // When navigator is open, intercept specific keys
      if (navigatorOpenRef?.current) {
        if (e.key === "Escape") {
          e.preventDefault();
          toggleNavigator?.();
          return;
        }
        const num = parseInt(e.key, 10);
        if (num >= 1 && num <= 9) {
          e.preventDefault();
          const keys = Array.from(parentGroups.keys());
          if (num <= keys.length) {
            const targetKey = keys[num - 1];
            if (targetKey) jumpToParent(targetKey);
            toggleNavigator?.();
          }
          return;
        }
        // Swallow other keys while navigator is open
        return;
      }

      switch (e.key) {
        case "ArrowRight":
          e.preventDefault();
          void advance();
          break;
        case "ArrowLeft":
          e.preventDefault();
          goBack();
          break;
        case "Enter":
          e.preventDefault();
          void advance();
          break;
        case "g":
        case "G":
          e.preventDefault();
          toggleNavigator?.();
          break;
        case "s":
        case "S":
          e.preventDefault();
          snoozeParent();
          break;
        case "Escape":
          e.preventDefault();
          onBack();
          break;
        default:
          break;
      }
    }

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [phase, current, advance, goBack, snoozeParent, onBack, navigatorOpenRef, toggleNavigator, parentGroups, jumpToParent]);

  return {
    phase,
    completionResults,
    current,
    answer,
    total,
    safeIndex,
    savingId,
    saveError,
    containerRef,
    updateAnswer,
    advance,
    goBack,
    skip,
    snoozeParent,
    parentGroups,
    jumpToParent,
    snoozeSpecificParent,
    localAnswers,
    skippedIds,
    onComplete,
  };
}
