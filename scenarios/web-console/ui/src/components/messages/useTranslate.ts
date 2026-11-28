import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import type { Translate } from "./workSummary";

/**
 * `t` narrowed to string keys with params: the generated catalog keys are
 * plain strings, and the row helpers (workSummary, noticeSummary) take this
 * shape so they stay pure and testable.
 */
export function useTranslate(): Translate {
  const { t } = useTranslation();
  return useCallback<Translate>((key, params) => t(key as never, params), [t]);
}
