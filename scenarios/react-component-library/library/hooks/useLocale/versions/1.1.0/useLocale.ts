/**
 * @libraryId react-component-library:useLocale
 * @displayName useLocale
 * @description
 * @version 1.1.0
 * @tags ["runtime","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-locale */

import { createContext, createElement, useContext, type ReactNode } from "react";

export type StringDefaults = Readonly<Record<string, string>>;
export interface DefinedStrings {
  namespace: string;
  defaults: StringDefaults;
}
export interface LibraryStringsProviderProps {
  children: ReactNode;
  strings?: DefinedStrings | readonly DefinedStrings[];
  translate?: (key: string, fallback: string) => string;
}

interface StringsContextValue {
  translate: (key: string, fallback: string) => string;
}

const StringsContext = createContext<StringsContextValue | null>(null);

export function defineStrings(namespace: string, defaults: StringDefaults): DefinedStrings {
  return { namespace, defaults };
}

function flattenStrings(
  strings: DefinedStrings | readonly DefinedStrings[] | undefined,
): StringDefaults {
  const entries = Array.isArray(strings) ? strings : strings ? [strings] : [];
  return Object.fromEntries(entries.flatMap((entry) => Object.entries(entry.defaults)));
}

export function LibraryStringsProvider({
  children,
  strings,
  translate,
}: LibraryStringsProviderProps) {
  const defaults = flattenStrings(strings);
  const value: StringsContextValue = {
    translate: (key, fallback) =>
      translate?.(key, defaults[key] ?? fallback) ?? defaults[key] ?? fallback,
  };
  return createElement(StringsContext.Provider, { value }, children);
}

export function useLocale() {
  return typeof document !== "undefined" ? document.documentElement.lang || "en" : "en";
}

export function useStrings(): (key: string, fallback: string) => string;
export function useStrings(key: string, fallback: string): string;
export function useStrings(
  key?: string,
  fallback?: string,
): ((key: string, fallback: string) => string) | string {
  const context = useContext(StringsContext);
  const resolver = (nextKey: string, nextFallback: string) =>
    context?.translate(nextKey, nextFallback) ?? nextFallback;
  return key === undefined ? resolver : resolver(key, fallback ?? "");
}
