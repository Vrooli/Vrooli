import { type KeyTree, type LocaleMessages } from "./catalog.js";
import { en } from "./locales/en.js";

function keyTree(node: LocaleMessages, prefix: string): LocaleMessages {
  return Object.fromEntries(
    Object.entries(node).map(([key, value]) => {
      const path = prefix ? `${prefix}.${key}` : key;
      return [key, typeof value === "string" ? path : keyTree(value, path)];
    }),
  );
}

/**
 * Typed key registry: `strings.viewer.close` is the literal `"viewer.close"`.
 * A misspelled or removed key is a compile error, and `t(strings.x.y)` only
 * accepts keys the English catalog really has.
 */
// The tree is built from `en` itself, so its runtime shape is exactly KeyTree<typeof en>.
export const strings = keyTree(en, "") as unknown as KeyTree<typeof en>;
