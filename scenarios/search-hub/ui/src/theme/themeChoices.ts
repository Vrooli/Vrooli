import { strings } from "../consts/strings";
import type { ThemeChoice } from "./ThemeProvider";

/**
 * Canonical theme-choice list shared by the TopBar switcher and the Settings
 * page so the two surfaces never drift. Each entry pairs the `ThemeChoice`
 * value with its i18n label *key path* — referencing
 * `strings.theme.choice.{light,dark,system}` statically here (rather than via a
 * dynamic `strings.theme.choice[c]` index at the call site) keeps the catalog
 * keys visible to the `strings/no-unused-keys` lint rule, which scans for
 * literal `strings.x.y` accessors.
 */
export const THEME_CHOICES = [
  { value: "light", labelKey: strings.theme.choice.light },
  { value: "dark", labelKey: strings.theme.choice.dark },
  { value: "system", labelKey: strings.theme.choice.system },
] as const satisfies readonly { value: ThemeChoice; labelKey: string }[];
