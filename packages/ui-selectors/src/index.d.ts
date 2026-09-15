export type LiteralSelectorTree = { readonly [key: string]: string | LiteralSelectorTree };
export type LiteralNode = string | LiteralSelectorTree;

export type ParamType = "string" | "number" | "enum";

export type ParamDefinition =
  | { readonly type: "string" }
  | { readonly type: "number" }
  | { readonly type: "enum"; readonly values: readonly (string | number)[] };

export type ParamSchema = Readonly<Record<string, ParamDefinition>>;

export type ParamValueType<T extends ParamDefinition> = T extends { type: "number" }
  ? number
  : T extends { type: "enum"; values: readonly (infer V)[] }
  ? V
  : string;

export type ParamValues<P extends ParamSchema | undefined> = P extends ParamSchema
  ? { [K in keyof P]: ParamValueType<P[K]> }
  : Record<string, never>;

export interface DynamicSelectorDefinition<P extends ParamSchema | undefined = undefined> {
  readonly kind: "dynamic-selector";
  readonly description: string;
  readonly params?: P;
  readonly testIdPattern?: string;
  readonly selectorPattern?: string;
}

export type DynamicSelectorBranch = {
  readonly [key: string]:
    | DynamicSelectorBranch
    | DynamicSelectorDefinition<ParamSchema | undefined>;
};

export type DynamicSelectorTree = DynamicSelectorBranch;

export type DynamicSelectorFn<P extends ParamSchema | undefined> = keyof ParamValues<P> extends never
  ? () => string
  : (params: ParamValues<P>) => string;

export type DynamicBranchResult<D extends DynamicSelectorTree> = {
  [K in keyof D]: D[K] extends DynamicSelectorDefinition<infer P>
  ? DynamicSelectorFn<P>
  : D[K] extends DynamicSelectorTree
  ? DynamicBranchResult<D[K]>
  : never;
};

export type SelectorTreeResult<L extends LiteralSelectorTree, D extends DynamicSelectorTree> = {
 [K in keyof L | keyof D]: K extends keyof L
  ? L[K] extends string ? string : SelectorTreeResult<Extract<L[K],LiteralSelectorTree>, K extends keyof D ? Extract<D[K],DynamicSelectorTree> : {}>
  : K extends keyof D ? D[K] extends DynamicSelectorDefinition<infer P> ? DynamicSelectorFn<P> : D[K] extends DynamicSelectorTree ? SelectorTreeResult<{},D[K]> : never : never;
};

export interface SelectorManifest {
 schemaVersion?: 1;
 selectors: Record<string, {testId: string; selector: string}>;
 dynamicSelectors: Record<string, {description: string; selectorPattern: string; testIdPattern?: string; params: Array<{name: string; type: ParamType; values?: readonly (string | number)[]}>}>;
}
export declare function defineDynamicSelector<P extends ParamSchema | undefined>(definition: Omit<DynamicSelectorDefinition<P>, 'kind'>): DynamicSelectorDefinition<P>;
export declare function createSelectorRegistry<L extends LiteralSelectorTree, D extends DynamicSelectorTree>(literalTree: L, dynamicTree: D): {selectors: SelectorTreeResult<L,D>; manifest: SelectorManifest};
export declare function createSelectorRegistry<L extends LiteralSelectorTree, D extends DynamicSelectorTree, B extends LiteralSelectorTree>(literalTree: L, dynamicTree: D, library: B): {selectors: SelectorTreeResult<L,D> & {library: B}; manifest: SelectorManifest};
export declare function resolveSelector(manifest: SelectorManifest, key: string, params?: Record<string,string|number>): string;
export declare function scopeSelector(parent: string, child: string): string;
export declare function escapeCSSValue(value: string | number): string;
