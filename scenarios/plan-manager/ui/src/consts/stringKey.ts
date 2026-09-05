import { type Strings } from "./strings";

/**
 * StringKey is the union of every leaf key-path value in the `strings`
 * registry — i.e. exactly the set of arguments `t()` accepts. Use it to type a
 * `labelKey` field or a `Record<Enum, StringKey>` map so the typed `t()` call
 * stays valid without widening to plain `string`.
 */
type Leaves<T> = T extends string ? T : T extends object ? { [K in keyof T]: Leaves<T[K]> }[keyof T] : never;

export type StringKey = Leaves<Strings>;
