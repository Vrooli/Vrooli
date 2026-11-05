import en from "./locales/en.json";

type TranslationOptions = { defaultValue?: string };
type TranslationTree = { [key: string]: string | TranslationTree };

/**
 * Library strings arrive as dotted keys ("feedback.toast.dismiss-notification") but the
 * locale file nests them, because that is the shape the library's own tooling writes.
 * Flattening once at load keeps the lookup a single map read and means a nested entry
 * resolves rather than silently falling through to its English default.
 */
function flatten(tree: TranslationTree, prefix = "", out: Record<string, string> = {}) {
  for (const [key, value] of Object.entries(tree)) {
    if (key.startsWith("$")) continue;
    const path = prefix ? `${prefix}.${key}` : key;
    if (typeof value === "string") {
      out[path] = value;
    } else {
      flatten(value, path, out);
    }
  }
  return out;
}

const translations = flatten(en as TranslationTree);

/** Minimal local translator used by the RCL library-string bridge. */
export const i18n = {
  t(key: string, options?: TranslationOptions): string {
    return translations[key] ?? options?.defaultValue ?? key;
  },
};
