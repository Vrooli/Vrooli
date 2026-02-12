/**
 * Hook for auto-suggesting skills based on user input and conversation context.
 *
 * Trigger model:
 * - Primary trigger: inputText changes (debounced 2s after last keystroke)
 * - Guards: typing debounce (2s), in-flight suppression, throttle (30s unless text changed >20%)
 * - Returns [] when disabled, no inputText, or on error
 */

import { useState, useEffect, useRef, useCallback } from "react";
import { fetchSkillSuggestions, type SuggestedSkill } from "@/lib/api";

const DEBOUNCE_MS = 2000;
const THROTTLE_MS = 30000;
const MIN_INPUT_LENGTH = 10;
const TEXT_CHANGE_THRESHOLD = 0.2; // 20% change required to bypass throttle

interface UseAutoSuggestSkillsOptions {
  chatId: string | undefined;
  inputText: string;
  selectedSkillIds: string[];
  enabled: boolean;
}

interface UseAutoSuggestSkillsReturn {
  suggestions: SuggestedSkill[];
  isLoading: boolean;
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
}: UseAutoSuggestSkillsOptions): UseAutoSuggestSkillsReturn {
  const [suggestions, setSuggestions] = useState<SuggestedSkill[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [dismissedIds, setDismissedIds] = useState<Set<string>>(new Set());

  // Refs for debounce/throttle/in-flight state
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout>>();
  const lastFetchTimeRef = useRef(0);
  const lastFetchTextRef = useRef("");
  const inFlightRef = useRef(false);
  const abortControllerRef = useRef<AbortController>();
  const prevChatIdRef = useRef(chatId);

  // Reset dismissed IDs when chat changes
  useEffect(() => {
    if (prevChatIdRef.current !== chatId) {
      setDismissedIds(new Set());
      setSuggestions([]);
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
    if (!enabled || inputText.length < MIN_INPUT_LENGTH) {
      return;
    }

    debounceTimerRef.current = setTimeout(() => {
      // Guard: in-flight suppression
      if (inFlightRef.current) return;

      // Guard: throttle (unless text changed significantly)
      const now = Date.now();
      const timeSinceLastFetch = now - lastFetchTimeRef.current;
      if (
        timeSinceLastFetch < THROTTLE_MS &&
        !textChangedSignificantly(lastFetchTextRef.current, inputText)
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
      })
        .then((resp) => {
          if (!controller.signal.aborted) {
            setSuggestions(resp.suggestions);
            lastFetchTimeRef.current = Date.now();
            lastFetchTextRef.current = inputText;
          }
        })
        .catch(() => {
          // Graceful degradation - just clear suggestions on error
          if (!controller.signal.aborted) {
            setSuggestions([]);
          }
        })
        .finally(() => {
          if (!controller.signal.aborted) {
            inFlightRef.current = false;
            setIsLoading(false);
          }
        });
    }, DEBOUNCE_MS);

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [inputText, chatId, selectedSkillIds, dismissedIds, enabled]);

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
    return { suggestions: [], isLoading: false, dismiss, dismissAll };
  }

  return { suggestions, isLoading, dismiss, dismissAll };
}
