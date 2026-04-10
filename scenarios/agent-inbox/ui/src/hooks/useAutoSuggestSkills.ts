/**
 * Hook for auto-suggesting skills based on user input and conversation context.
 */

import { useState, useEffect, useRef, useCallback } from "react";
import { fetchSkillSuggestions, type SuggestedSkill } from "@/lib/api";

const DEFAULT_DEBOUNCE_MS = 900;
const DEFAULT_THROTTLE_MS = 10000;
const DEFAULT_MIN_INPUT_LENGTH = 10;
const TEXT_CHANGE_THRESHOLD = 0.2; // 20% change required to bypass throttle

interface UseAutoSuggestSkillsOptions {
  chatId: string | undefined;
  inputText: string;
  selectedSkillIds: string[];
  enabled: boolean;
  debounceMs?: number;
  throttleMs?: number;
  minInputLength?: number;
  minScorePercent?: number;
  maxSuggestions?: number;
}

interface UseAutoSuggestSkillsReturn {
  suggestions: SuggestedSkill[];
  isLoading: boolean;
  didSearch: boolean;
  dismiss: (skillId: string) => void;
  dismissAll: () => void;
}

function textChangedSignificantly(prev: string, next: string): boolean {
  if (!prev && next) return true;
  if (!next) return false;
  const maxLen = Math.max(prev.length, next.length);
  if (maxLen === 0) return false;
  const diff = Math.abs(prev.length - next.length);
  return diff / maxLen > TEXT_CHANGE_THRESHOLD;
}

export function useAutoSuggestSkills({
  chatId,
  inputText,
  selectedSkillIds,
  enabled,
  debounceMs = DEFAULT_DEBOUNCE_MS,
  throttleMs = DEFAULT_THROTTLE_MS,
  minInputLength = DEFAULT_MIN_INPUT_LENGTH,
  minScorePercent = 0,
  maxSuggestions = 5,
}: UseAutoSuggestSkillsOptions): UseAutoSuggestSkillsReturn {
  const [suggestions, setSuggestions] = useState<SuggestedSkill[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [didSearch, setDidSearch] = useState(false);
  const [dismissedIds, setDismissedIds] = useState<Set<string>>(new Set());

  // Refs for debounce/throttle/in-flight state
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout>>();
  const lastFetchTimeRef = useRef(0);
  const lastFetchTextRef = useRef("");
  const inFlightRef = useRef(false);
  const abortControllerRef = useRef<AbortController>();
  const prevChatIdRef = useRef(chatId);

  const normalizedMinScore = Math.max(0, Math.min(100, minScorePercent));
  const normalizedMaxSuggestions = Math.max(1, Math.min(20, maxSuggestions));

  // Reset state when chat changes
  useEffect(() => {
    if (prevChatIdRef.current !== chatId) {
      setDismissedIds(new Set());
      setSuggestions([]);
      setDidSearch(false);
      prevChatIdRef.current = chatId;
    }
  }, [chatId]);

  // Main effect: debounced fetch on input text changes
  useEffect(() => {
    // Clear any existing timer
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    // Guard: disabled or too short
    if (!enabled || inputText.trim().length < minInputLength) {
      setSuggestions([]);
      setDidSearch(false);
      return;
    }

    debounceTimerRef.current = setTimeout(() => {
      // Guard: in-flight suppression
      if (inFlightRef.current) return;

      // Guard: throttle (unless text changed significantly)
      const now = Date.now();
      const timeSinceLastFetch = now - lastFetchTimeRef.current;
      if (
        timeSinceLastFetch < throttleMs
        && !textChangedSignificantly(lastFetchTextRef.current, inputText)
      ) {
        return;
      }

      // Cancel any previous in-flight request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      const controller = new AbortController();
      abortControllerRef.current = controller;
      inFlightRef.current = true;
      setIsLoading(true);

      const excludeIds = [
        ...selectedSkillIds,
        ...Array.from(dismissedIds),
      ];

      fetchSkillSuggestions({
        inputText,
        chatId,
        excludeSkillIds: excludeIds.length > 0 ? excludeIds : undefined,
        signal: controller.signal,
      })
        .then((resp) => {
          if (!controller.signal.aborted) {
            const filtered = resp.suggestions
              .filter((skill) => skill.scorePercent >= normalizedMinScore)
              .slice(0, normalizedMaxSuggestions);
            setSuggestions(filtered);
            setDidSearch(true);
            lastFetchTimeRef.current = Date.now();
            lastFetchTextRef.current = inputText;
          }
        })
        .catch(() => {
          // Graceful degradation - just clear suggestions on error
          if (!controller.signal.aborted) {
            setSuggestions([]);
            setDidSearch(true);
          }
        })
        .finally(() => {
          if (!controller.signal.aborted) {
            inFlightRef.current = false;
            setIsLoading(false);
          }
        });
    }, debounceMs);

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [
    inputText,
    chatId,
    selectedSkillIds,
    dismissedIds,
    enabled,
    debounceMs,
    throttleMs,
    minInputLength,
    normalizedMinScore,
    normalizedMaxSuggestions,
  ]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  const dismiss = useCallback((skillId: string) => {
    setDismissedIds((prev) => new Set(prev).add(skillId));
    setSuggestions((prev) => prev.filter((s) => s.id !== skillId));
  }, []);

  const dismissAll = useCallback(() => {
    setDismissedIds((prev) => {
      const next = new Set(prev);
      suggestions.forEach((s) => next.add(s.id));
      return next;
    });
    setSuggestions([]);
  }, [suggestions]);

  // Return empty when disabled
  if (!enabled) {
    return { suggestions: [], isLoading: false, didSearch: false, dismiss, dismissAll };
  }

  return { suggestions, isLoading, didSearch, dismiss, dismissAll };
}
