/**
 * Vrooli Ascension selector registry
 *
 * This file is the single source of truth for every selector used by the UI and
 * by Vrooli Ascension workflows. We deliberately model selectors as two
 * declarative maps (one literal, one dynamic) and rely on a small helper to
 * produce the typed `selectors` export plus the manifest consumed by workflow
 * linting. Do not hand-roll selector helpers or change this structure—update the
 * maps below so UI code, automation flows, and the manifest builder all stay in
 * sync across every scenario.
 *
 * ## Auto-Generated Manifest
 *
 * The `selectors.manifest.json` file is automatically generated from this file
 * during the testing process. If you need to add or modify selectors:
 *
 * 1. Update the `literalSelectors` object below for static selectors
 * 2. Update the `dynamicSelectorDefinitions` object for parameterized selectors
 * 3. The manifest will be regenerated automatically when tests run
 *
 * DO NOT manually edit `selectors.manifest.json` - your changes will be overwritten!
 */

type LiteralSelectorTree = { readonly [key: string]: string | LiteralSelectorTree };
type LiteralNode = string | LiteralSelectorTree;

type ParamType = "string" | "number" | "enum";

type ParamDefinition =
  | { readonly type: "string" }
  | { readonly type: "number" }
  | { readonly type: "enum"; readonly values: readonly (string | number)[] };

type ParamSchema = Readonly<Record<string, ParamDefinition>>;

type ParamValueType<T extends ParamDefinition> = T extends { type: "number" }
  ? number
  : T extends { type: "enum"; values: readonly (infer V)[] }
  ? V
  : string;

type ParamValues<P extends ParamSchema | undefined> = P extends ParamSchema
  ? { [K in keyof P]: ParamValueType<P[K]> }
  : Record<string, never>;

interface DynamicSelectorDefinition<P extends ParamSchema | undefined = undefined> {
  readonly kind: "dynamic-selector";
  readonly description: string;
  readonly params?: P;
  readonly testIdPattern?: string;
  readonly selectorPattern?: string;
}

type DynamicSelectorBranch = {
  readonly [key: string]: DynamicSelectorBranch | DynamicSelectorDefinition<any>;
};

type DynamicSelectorTree = DynamicSelectorBranch;

type DynamicSelectorFn<P extends ParamSchema | undefined> = keyof ParamValues<P> extends never
  ? () => string
  : (params: ParamValues<P>) => string;

type DynamicBranchResult<D extends DynamicSelectorTree> = {
  [K in keyof D]: D[K] extends DynamicSelectorDefinition<infer P>
  ? DynamicSelectorFn<P>
  : D[K] extends DynamicSelectorTree
  ? DynamicBranchResult<D[K]>
  : never;
};

type SelectorTreeResult<
  L extends LiteralSelectorTree,
  D extends DynamicSelectorTree,
> = {
  [K in keyof L]: L[K] extends string
    ? string
    : SelectorTreeResult<
        Extract<L[K], LiteralSelectorTree>,
        K extends keyof D ? Extract<D[K], DynamicSelectorTree> : DynamicSelectorTree
      >;
} & (D extends DynamicSelectorTree ? DynamicBranchResult<D> : {});

const TEMPLATE_TOKEN = /\$\{([^}]+)\}/g;

const formatTemplate = (template: string, values: Record<string, string | number>, keyPath: string) =>
  template.replace(TEMPLATE_TOKEN, (_match, token) => {
    if (!(token in values)) {
      throw new Error(`Missing parameter '${token}' for selector '${keyPath}'`);
    }
    return String(values[token]);
  });

const toDataTestIdSelector = (testId: string) => `[data-testid="${testId}"]`;

const isDynamicDefinition = (value: unknown): value is DynamicSelectorDefinition<any> =>
  Boolean(value && typeof value === "object" && (value as DynamicSelectorDefinition).kind === "dynamic-selector");

const normalizeParams = (
  definition: DynamicSelectorDefinition<any>,
  raw: Record<string, string | number>,
  path: string,
) => {
  const schema = definition.params ?? ({} as ParamSchema);
  const normalized: Record<string, string | number> = {};

  for (const key of Object.keys(schema)) {
    if (!(key in raw)) {
      throw new Error(`Selector '${path}' is missing parameter '${key}'`);
    }
    const definitionEntry = schema[key];
    const value = raw[key];
    if (definitionEntry.type === "number") {
      if (typeof value !== "number") {
        throw new Error(`Selector '${path}' parameter '${key}' must be numeric`);
      }
      normalized[key] = value;
      continue;
    }
    if (definitionEntry.type === "enum") {
      if (!definitionEntry.values.includes(value)) {
        throw new Error(
          `Selector '${path}' parameter '${key}' must be one of: ${definitionEntry.values.join(", ")}`,
        );
      }
      normalized[key] = value;
      continue;
    }
    normalized[key] = value;
  }

  const extras = Object.keys(raw).filter((key) => !(key in schema));
  if (extras.length > 0) {
    throw new Error(`Selector '${path}' received unknown parameter(s): ${extras.join(", ")}`);
  }

  return normalized;
};

const flattenLiteralSelectors = (
  tree: LiteralSelectorTree,
  prefix: string[] = [],
  target: Record<string, { testId: string; selector: string }> = {},
) => {
  for (const [key, value] of Object.entries(tree)) {
    const nextPath = [...prefix, key];
    if (typeof value === "string") {
      const manifestKey = nextPath.join(".");
      target[manifestKey] = {
        testId: value,
        selector: toDataTestIdSelector(value),
      };
      continue;
    }
    flattenLiteralSelectors(value, nextPath, target);
  }
  return target;
};

const flattenDynamicSelectors = (
  tree: DynamicSelectorTree,
  prefix: string[] = [],
  target: Record<string, {
    description: string;
    selectorPattern: string;
    testIdPattern?: string;
    params: Array<{ name: string; type: ParamType; values?: readonly (string | number)[] }>;
  }> = {},
) => {
  for (const [key, value] of Object.entries(tree)) {
    const nextPath = [...prefix, key];
    if (isDynamicDefinition(value)) {
      const manifestKey = nextPath.join(".");
      const paramEntries = Object.entries(value.params ?? {}) as Array<[string, ParamDefinition]>;
      target[manifestKey] = {
        description: value.description,
        selectorPattern:
          value.selectorPattern ?? (value.testIdPattern ? toDataTestIdSelector(value.testIdPattern) : ""),
        testIdPattern: value.testIdPattern,
        params: paramEntries.map(([name, config]) => ({
          name,
          type: config.type,
          values: config.type === "enum" ? config.values : undefined,
        })),
      };
      continue;
    }
    flattenDynamicSelectors(value, nextPath, target);
  }
  return target;
};

const mergeLiteralAndDynamicNodes = (
  literalNode: LiteralSelectorTree | undefined,
  dynamicNode: DynamicSelectorTree | undefined,
  path: string[] = [],
): Record<string, unknown> => {
  const merged: Record<string, unknown> = {};
  const keys = new Set([
    ...Object.keys(literalNode ?? {}),
    ...Object.keys(dynamicNode ?? {}),
  ]);

  keys.forEach((key) => {
    const literalValue: LiteralNode | undefined = literalNode?.[key];
    const dynamicValue = dynamicNode?.[key];
    const nextPath = [...path, key];

    if (typeof literalValue === "string") {
      merged[key] = literalValue;
      return;
    }

    if (literalValue && typeof literalValue === "object") {
      merged[key] = mergeLiteralAndDynamicNodes(
        literalValue as LiteralSelectorTree,
        isDynamicDefinition(dynamicValue) ? undefined : (dynamicValue as DynamicSelectorTree | undefined),
        nextPath,
      );
      return;
    }

    if (dynamicValue) {
      if (isDynamicDefinition(dynamicValue)) {
        merged[key] = createDynamicSelectorFn(dynamicValue, nextPath.join("."));
        return;
      }
      merged[key] = mergeLiteralAndDynamicNodes(undefined, dynamicValue as DynamicSelectorTree, nextPath);
    }
  });

  return merged;
};

const createDynamicSelectorFn = (
  definition: DynamicSelectorDefinition<any>,
  path: string,
) => {
  return (params?: Record<string, string | number>) => {
    const normalized = normalizeParams(definition, params ?? {}, path);
    const template = definition.testIdPattern ?? definition.selectorPattern;
    if (!template) {
      throw new Error(`Selector '${path}' is missing both testIdPattern and selectorPattern`);
    }
    return formatTemplate(template, normalized, path);
  };
};

const defineDynamicSelector = <P extends ParamSchema | undefined>(
  definition: Omit<DynamicSelectorDefinition<P>, "kind">,
): DynamicSelectorDefinition<P> => ({
  ...definition,
  kind: "dynamic-selector",
});

const createSelectorRegistry = <
  L extends LiteralSelectorTree,
  D extends DynamicSelectorTree,
>(literalTree: L, dynamicTree: D) => {
  const selectors = mergeLiteralAndDynamicNodes(literalTree, dynamicTree) as SelectorTreeResult<L, D>;
  const manifest = {
    selectors: flattenLiteralSelectors(literalTree),
    dynamicSelectors: flattenDynamicSelectors(dynamicTree),
  };
  return { selectors, manifest };
};

const literalSelectors: LiteralSelectorTree = {
  admin: {
    login: {
      email: 'admin-login-email',
      passwordField: 'admin-login-password',
      submit: 'admin-login-submit',
      error: 'admin-login-error',
    },
    breadcrumb: 'admin-breadcrumb',
    nav: {
      home: 'nav-home',
      analytics: 'nav-analytics',
      customization: 'nav-customization',
      logout: 'nav-logout',
    },
    home: {
      resumePanel: 'admin-resume-panel',
      resumeCustomization: 'admin-resume-customization',
      resumeAnalytics: 'admin-resume-analytics',
      resumeCard: 'admin-resume-card',
      resumeAnalyticsCard: 'admin-resume-analytics-card',
      experienceGuide: 'admin-experience-guide',
      guideAnalytics: 'admin-guide-analytics',
      guideCustomization: 'admin-guide-customization',
      guidePreview: 'admin-guide-preview',
    },
    mode: {
      analytics: 'admin-mode-analytics',
      customization: 'admin-mode-customization',
    },
    analytics: {
      filters: 'analytics-filters',
      timeRange: 'analytics-time-range',
      variantFilter: 'analytics-variant-filter',
      focusBanner: 'analytics-focus-banner',
      resetFilters: 'analytics-reset-filters',
      focusCustomize: 'analytics-focus-customize',
      focusPreview: 'analytics-focus-preview',
      totalVisitors: 'analytics-total-visitors',
      conversionRate: 'analytics-conversion-rate',
      topCta: 'analytics-top-cta',
      variantPerformance: 'analytics-variant-performance',
      variantDetail: 'analytics-variant-detail',
      variantActions: 'analytics-variant-actions',
    },
    customization: {
      triggerAgent: 'trigger-agent-customization',
      createVariant: 'create-variant',
      addSection: 'add-section',
      filterBar: 'variant-filter-bar',
      filterSearch: 'variant-search-input',
      filterAttentionToggle: 'variant-attention-filter',
      clearFilters: 'clear-variant-filters',
      needsAttentionFocus: 'needs-attention-focus',
      variantListSummary: 'variant-list-summary',
    },
    variant: {
      nameInput: 'variant-name-input',
      slugInput: 'variant-slug-input',
      descriptionInput: 'variant-description-input',
      weightInput: 'variant-weight-input',
      save: 'save-variant',
    },
    section: {
      form: 'section-form',
      preview: 'section-preview',
      typeInput: 'section-type-input',
      enabledInput: 'section-enabled-input',
      orderInput: 'section-order-input',
      save: 'save-section',
      content: {
        titleInput: 'content-title-input',
        subtitleInput: 'content-subtitle-input',
        ctaTextInput: 'content-cta-text-input',
        ctaUrlInput: 'content-cta-url-input',
        imageUrlInput: 'content-image-url-input',
      },
    },
    agent: {
      briefInput: 'agent-brief-input',
      assetsInput: 'agent-assets-input',
      previewInput: 'agent-preview-input',
      submit: 'agent-submit',
    },
  },
  publicLanding: {
    experienceHeader: 'landing-experience-header',
    navCta: 'landing-nav-cta',
    navMobile: 'landing-nav-mobile',
    navDownload: 'landing-nav-download',
  },
};

const dynamicSelectorDefinitions: DynamicSelectorTree = {
  admin: {
    analytics: {
      variantRow: defineDynamicSelector({
        description: 'Analytics variant performance row by variant ID',
        testIdPattern: 'analytics-variant-row-${id}',
        params: { id: { type: 'number' } },
      }),
      viewDetails: defineDynamicSelector({
        description: 'View details button for specific variant',
        testIdPattern: 'analytics-view-details-${id}',
        params: { id: { type: 'number' } },
      }),
      editVariant: defineDynamicSelector({
        description: 'Customize button for specific variant from analytics table',
        testIdPattern: 'analytics-edit-${id}',
        params: { id: { type: 'number' } },
      }),
    },
    customization: {
      variantCard: defineDynamicSelector({
        description: 'Variant card by slug',
        testIdPattern: 'variant-card-${slug}',
        params: { slug: { type: 'string' } },
      }),
      editVariant: defineDynamicSelector({
        description: 'Edit variant button by slug',
        testIdPattern: 'edit-variant-${slug}',
        params: { slug: { type: 'string' } },
      }),
      previewVariant: defineDynamicSelector({
        description: 'Preview variant button by slug',
        testIdPattern: 'preview-variant-${slug}',
        params: { slug: { type: 'string' } },
      }),
      archiveVariant: defineDynamicSelector({
        description: 'Archive variant button by slug',
        testIdPattern: 'archive-variant-${slug}',
        params: { slug: { type: 'string' } },
      }),
      variantAnalytics: defineDynamicSelector({
        description: 'Link to view analytics for a variant card',
        testIdPattern: 'variant-analytics-${slug}',
        params: { slug: { type: 'string' } },
      }),
      variantPerformance: defineDynamicSelector({
        description: 'Variant performance summary block',
        testIdPattern: 'variant-performance-${slug}',
        params: { slug: { type: 'string' } },
      }),
      variantStatus: defineDynamicSelector({
        description: 'Variant status badges block',
        testIdPattern: 'variant-status-${slug}',
        params: { slug: { type: 'string' } },
      }),
      deleteVariant: defineDynamicSelector({
        description: 'Delete variant button by slug',
        testIdPattern: 'delete-variant-${slug}',
        params: { slug: { type: 'string' } },
      }),
      section: defineDynamicSelector({
        description: 'Section item by ID',
        testIdPattern: 'section-${id}',
        params: { id: { type: 'number' } },
      }),
      editSection: defineDynamicSelector({
        description: 'Edit section button by ID',
        testIdPattern: 'edit-section-${id}',
        params: { id: { type: 'number' } },
      }),
    },
    breadcrumb: defineDynamicSelector({
      description: 'Breadcrumb segment by index',
      testIdPattern: 'breadcrumb-${index}',
      params: { index: { type: 'number' } },
    }),
  },
};

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
