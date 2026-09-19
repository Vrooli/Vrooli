import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
import type { GrantEffect } from "../../api/machines";

/** The effect families, in operator wording. */
export function useEffectLabel() {
  const { t } = useTranslation();
  return (effect: GrantEffect) =>
    t(
      effect === "read"
        ? strings.machines.effectRead
        : effect === "write"
          ? strings.machines.effectWrite
          : strings.machines.effectDestructive,
    );
}

/**
 * How far a grant reaches. A wildcard reaches apps that do not exist yet, so it
 * is never reported as a count — a number would understate it.
 */
export function useBreadthLabel() {
  const { t } = useTranslation();
  return (appCount: number, coversAllApps: boolean) =>
    coversAllApps ? t(strings.machines.everyApp) : t(strings.machines.appCount, { count: appCount });
}
