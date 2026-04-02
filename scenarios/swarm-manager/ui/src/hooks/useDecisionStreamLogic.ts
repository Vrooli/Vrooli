import { useState, useCallback, useRef, useEffect } from "react";
import { useBacklogStore } from "../stores/backlog-store";
import { backlogService } from "../services/backlog-service";
import { OTHER_KEY, parseWorkshopRound, buildWorkshopRoundContent } from "../lib/workshop-files";
import type { QuestionAnswer } from "../components/backlog/question-renderers";
import type { CrossItemQuestion } from "../lib/command-post-utils";
import type { BacklogItem, BacklogKind } from "../types";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface DecisionStreamResults {
  answeredCount: number;
  skippedCount: number;
  snoozedCount: number;
  unlockedItems: { kind: BacklogKind; name: string; title: string; action: "finalize" | "run" }[];
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function groupByParent(questions: CrossItemQuestion[]): Map<string, CrossItemQuestion[]> {
  const map = new Map<string, CrossItemQuestion[]>();
  for (const ciq of questions) {
    const key = `${ciq.parentKind}/${ciq.parentName}`;
    const list = map.get(key);
    if (list) list.push(ciq);
    else map.set(key, [ciq]);
  }
  return map;
}

function snoozeKey(kind: BacklogKind, name: string): string {
  return `backlog:${kind}/${name}`;
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export interface UseDecisionStreamLogicArgs {
  questions: CrossItemQuestion[];
  onComplete: (results: DecisionStreamResults) => void;
  onBack: () => void;
  onSnoozeItem: (key: string) => void;
}

export function useDecisionStreamLogic({
  questions,
  onComplete,
  onBack,
  onSnoozeItem,
}: UseDecisionStreamLogicArgs) {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [localAnswers, setLocalAnswers] = useState<Map<string, QuestionAnswer>>(() => new Map());
  const [skippedIds, setSkippedIds] = useState<Set<string>>(() => new Set());
  const [snoozedItemKeys, setSnoozedItemKeys] = useState<Set<string>>(() => new Set());
  const [savingId, setSavingId] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [phase, setPhase] = useState<"answering" | "completing">("answering");
  const [contextExpanded, setContextExpanded] = useState(false);
  const [descExpanded, setDescExpanded] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const prevParentRef = useRef<string>("");

  const backlogItems = useBacklogStore((s) => s.items);

  const activeQuestions = questions.filter(
    (ciq) => !snoozedItemKeys.has(snoozeKey(ciq.parentKind, ciq.parentName)),
  );
  const total = activeQuestions.length;
  const safeIndex = Math.min(currentIndex, Math.max(0, total - 1));
  const current = activeQuestions[safeIndex] as CrossItemQuestion | undefined;
  const answer = current ? localAnswers.get(current.question.id) : undefined;

  const storeItem = current
    ? backlogItems.find((i) => i.kind === current.parentKind && i.name === current.parentName)
    : undefined;

  const [fetchedItems, setFetchedItems] = useState<Map<string, BacklogItem>>(() => new Map());
  const fetchingRef = useRef<Set<string>>(new Set());

  const parentItemKey = current ? `${current.parentKind}/${current.parentName}` : "";
  const parentItem = storeItem ?? fetchedItems.get(parentItemKey);

  useEffect(() => {
    if (!current || storeItem || fetchedItems.has(parentItemKey) || fetchingRef.current.has(parentItemKey)) return;
    fetchingRef.current.add(parentItemKey);
    void backlogService.get(current.parentKind, current.parentName)
      .then((item) => {
        setFetchedItems((prev) => {
          const next = new Map(prev);
          next.set(parentItemKey, item);
          return next;
        });
      })
      .catch(() => {
        // Item may be deleted — leave as unavailable
      })
      .finally(() => {
        fetchingRef.current.delete(parentItemKey);
      });
  }, [current, storeItem, parentItemKey, fetchedItems]);

  useEffect(() => {
    const parentKey = current ? `${current.parentKind}/${current.parentName}` : "";
    if (parentKey !== prevParentRef.current) {
      setContextExpanded(false);
      setDescExpanded(false);
      prevParentRef.current = parentKey;
    }
  }, [current]);

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
  ) => {
    if (!a) return;
    const q = ciq.question;
    setSavingId(q.id);
    setSaveError(null);
    try {
      if (q.source === "workshop" && q.round_number != null && a.selected?.trim()) {
        const roundNum = String(q.round_number).padStart(3, "0");
        const filePath = `workshop/round-${roundNum}.json`;
        const content = await backlogService.getFileContent(ciq.parentKind, ciq.parentName, filePath);
        const parsed = parseWorkshopRound(content);
        if (parsed.round) {
          const round = parsed.round;
          const item = (round.items ?? []).find((i) => i.id === q.id);
          if (item) {
            item.selected = a.selected === OTHER_KEY ? OTHER_KEY : a.selected;
            item.freeform = a.selected === OTHER_KEY ? (a.freeform ?? null) : null;
            item.notes = a.notes ?? null;
          }
          await backlogService.saveFileContent(
            ciq.parentKind, ciq.parentName, filePath,
            buildWorkshopRoundContent(round), "application/json",
          );
        }
      } else if (q.source === "review" && (a.reviewStatus === "approved" || a.reviewStatus === "flagged")) {
        await backlogService.batchReview(ciq.parentKind, ciq.parentName, [{
          id: q.id,
          type: q.review_type ?? "target",
          module_id: q.module_id,
          review_status: a.reviewStatus,
          review_comment: a.reviewComment ?? "",
        }]);
      }
    } catch {
      setSaveError("Save failed — will retry on next advance");
    } finally {
      setSavingId(null);
    }
  }, []);

  // ---------------------------------------------------------------------------
  // Completion
  // ---------------------------------------------------------------------------

  const handleCompletion = useCallback(async () => {
    setPhase("completing");

    const answeredCount = Array.from(localAnswers.values()).filter((a) => {
      return (a.selected?.trim()) || a.reviewStatus === "approved" || a.reviewStatus === "flagged";
    }).length;

    const parentGroups = groupByParent(activeQuestions);
    const unlockedItems: DecisionStreamResults["unlockedItems"] = [];

    for (const [parentKey, groupQuestions] of parentGroups) {
      const workshopQ = groupQuestions.find(
        (ciq) => ciq.question.source === "workshop" && ciq.question.round_number != null,
      );
      if (!workshopQ || workshopQ.question.round_number == null) continue;

      const hasAnswers = groupQuestions.some((ciq) => {
        const a = localAnswers.get(ciq.question.id);
        return a && (a.selected?.trim() || a.reviewStatus === "approved" || a.reviewStatus === "flagged");
      });
      if (!hasAnswers) continue;

      try {
        const roundNumber = workshopQ.question.round_number;
        const roundNum = String(roundNumber).padStart(3, "0");
        const filePath = `workshop/round-${roundNum}.json`;
        const content = await backlogService.getFileContent(
          workshopQ.parentKind, workshopQ.parentName, filePath,
        );
        const result = await backlogService.workshopSave(
          workshopQ.parentKind, workshopQ.parentName, roundNumber, content,
        );
        if (result.autoAdvance?.triggered) {
          const [, name] = parentKey.split("/");
          unlockedItems.push({
            kind: workshopQ.parentKind,
            name: name ?? workshopQ.parentName,
            title: workshopQ.parentTitle,
            action: result.autoAdvance.nextMode === "finalize" ? "finalize" : "run",
          });
        }
      } catch {
        // Non-fatal: continue with other items
      }
    }

    onComplete({
      answeredCount,
      skippedCount: skippedIds.size,
      snoozedCount: snoozedItemKeys.size,
      unlockedItems,
    });
  }, [activeQuestions, localAnswers, skippedIds, snoozedItemKeys, onComplete]);

  // ---------------------------------------------------------------------------
  // Navigation
  // ---------------------------------------------------------------------------

  const isAllDone = useCallback(() => {
    return activeQuestions.every((ciq) => {
      if (skippedIds.has(ciq.question.id)) return true;
      const a = localAnswers.get(ciq.question.id);
      if (!a) return false;
      if (ciq.question.source === "workshop") return !!a.selected?.trim();
      return a.reviewStatus === "approved" || a.reviewStatus === "flagged";
    });
  }, [activeQuestions, skippedIds, localAnswers]);

  const advance = useCallback(async () => {
    if (!current) return;
    const a = localAnswers.get(current.question.id);
    if (a) {
      await saveAnswer(current, a);
    }
    if (safeIndex < total - 1) {
      setCurrentIndex(safeIndex + 1);
    }
    if (isAllDone()) {
      handleCompletion();
    }
  }, [current, safeIndex, total, localAnswers, saveAnswer, isAllDone, handleCompletion]);

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
      if (ciq.question.source === "workshop") return !!a.selected?.trim();
      return a.reviewStatus === "approved" || a.reviewStatus === "flagged";
    });
    if (allDone) handleCompletion();
  }, [current, safeIndex, total, skippedIds, activeQuestions, localAnswers, handleCompletion]);

  const snoozeParent = useCallback(() => {
    if (!current) return;
    const key = snoozeKey(current.parentKind, current.parentName);
    const newSnoozed = new Set(snoozedItemKeys);
    newSnoozed.add(key);
    setSnoozedItemKeys(newSnoozed);
    onSnoozeItem(key);
  }, [current, snoozedItemKeys, onSnoozeItem]);

  // ---------------------------------------------------------------------------
  // Keyboard shortcuts
  // ---------------------------------------------------------------------------

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      const tag = (e.target as HTMLElement)?.tagName;
      if (tag === "TEXTAREA" || tag === "INPUT") return;
      if (phase !== "answering" || !current) return;

      switch (e.key) {
        case "ArrowRight":
          e.preventDefault();
          advance();
          break;
        case "ArrowLeft":
          e.preventDefault();
          goBack();
          break;
        case "Enter":
          e.preventDefault();
          advance();
          break;
        case "s":
        case "S":
          e.preventDefault();
          snoozeParent();
          break;
        case "i":
        case "I":
          e.preventDefault();
          setContextExpanded((prev) => !prev);
          break;
        case "Escape":
          e.preventDefault();
          if (contextExpanded) {
            setContextExpanded(false);
          } else {
            onBack();
          }
          break;
        default: {
          const num = parseInt(e.key, 10);
          if (num >= 1 && num <= 9 && current.question.source === "workshop") {
            e.preventDefault();
            const options = current.question.options ?? [];
            if (num <= options.length) {
              const opt = options[num - 1];
              if (opt) {
                updateAnswer(current.question.id, {
                  selected: opt.key,
                  freeform: undefined,
                });
              }
            } else if (num === options.length + 1) {
              updateAnswer(current.question.id, { selected: OTHER_KEY });
            }
          }
          break;
        }
      }
    }

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [phase, current, advance, goBack, snoozeParent, onBack, updateAnswer, contextExpanded]);

  return {
    phase,
    current,
    answer,
    parentItem,
    total,
    safeIndex,
    savingId,
    saveError,
    contextExpanded,
    setContextExpanded,
    descExpanded,
    setDescExpanded,
    containerRef,
    updateAnswer,
    advance,
    goBack,
    skip,
    snoozeParent,
  };
}
