import { en } from "./locales/en.js";

/** A catalog of copy: nested objects whose leaves are strings. */
export interface LocaleMessages { [key: string]: string | LocaleMessages }

/** The dotted paths of every string leaf of a catalog type. */
type KeyPaths<T, Prefix extends string = ""> = {
  [K in keyof T & string]: T[K] extends string ? `${Prefix}${K}` : KeyPaths<T[K], `${Prefix}${K}.`>;
}[keyof T & string];

/** Every key the viewer can render, derived from the English catalog. */
export type StringKey = KeyPaths<typeof en>;

/** The English catalog's shape with each leaf replaced by its dotted key path. */
export type KeyTree<T, Prefix extends string = ""> = {
  readonly [K in keyof T & string]: T[K] extends string ? `${Prefix}${K}` : KeyTree<T[K], `${Prefix}${K}.`>;
};

/** Overrides a host may supply: any subset of the keys, typed against the catalog. */
export type MessageOverrides = Partial<Record<StringKey, string>>;
