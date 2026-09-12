const TEMPLATE_TOKEN = /\$\{([^}]+)\}/g;
const formatTemplate = (template, values, keyPath, css = false) => template.replace(TEMPLATE_TOKEN, (_match, token) => {
    if (!Object.hasOwn(values, token)) {
        throw new Error(`Missing parameter '${token}' for selector '${keyPath}'`);
    }
    return css ? escapeCSSValue(values[token]) : String(values[token]);
});
// Hex escapes work in both quoted attribute values and CSS identifiers.
export const escapeCSSValue = (value) => String(value).replace(/[^a-zA-Z0-9_-]/gu, char => {
    if (char === '\0') throw new Error('Selector values must not contain NUL');
    return `\\${char.codePointAt(0).toString(16)} `;
});
// Quoted attribute values need fewer escapes than CSS identifiers. Preserve
// readable DOM IDs in manifests while still handling quotes and control bytes.
const escapeCSSString = (value) => String(value).replace(/["\\\u0000-\u001f\u007f]/gu, char => escapeCSSValue(char));
const toDataTestIdSelector = (testId) => `[data-testid="${escapeCSSString(testId)}"]`;
const testIdPatternSelector = (pattern) => `[data-testid="${pattern.split(/(\$\{[^}]+\})/g).map(part => /^\$\{[^}]+\}$/.test(part) ? part : escapeCSSString(part)).join('')}"]`;
const isDynamicDefinition = (value) => Boolean(value &&
    typeof value === "object" &&
    value.kind === "dynamic-selector");
const normalizeParams = (definition, raw, path) => {
    const schema = definition.params ?? {};
    const normalized = {};
    for (const [key, definitionEntry] of Object.entries(schema)) {
        if (!Object.hasOwn(raw, key)) {
            throw new Error(`Selector '${path}' is missing parameter '${key}'`);
        }
        const value = raw[key];
        if (value === undefined) {
            throw new Error(`Selector '${path}' parameter '${key}' is undefined`);
        }
        if (definitionEntry.type === "number") {
            if (typeof value !== "number" || !Number.isFinite(value)) {
                throw new Error(`Selector '${path}' parameter '${key}' must be numeric`);
            }
            normalized[key] = value;
            continue;
        }
        if (definitionEntry.type === "enum") {
            if (!definitionEntry.values.includes(value)) {
                throw new Error(`Selector '${path}' parameter '${key}' must be one of: ${definitionEntry.values.join(", ")}`);
            }
            normalized[key] = value;
            continue;
        }
        if (definitionEntry.type !== 'string' || typeof value !== 'string') {
            throw new Error(`Selector '${path}' parameter '${key}' must be a string`);
        }
        normalized[key] = value;
    }
    const extras = Object.keys(raw).filter((key) => !Object.hasOwn(schema, key));
    if (extras.length > 0) {
        throw new Error(`Selector '${path}' received unknown parameter(s): ${extras.join(", ")}`);
    }
    return normalized;
};
const flattenLiteralSelectors = (tree, prefix = [], target = {}) => {
    for (const [key, value] of Object.entries(tree)) {
        const nextPath = [...prefix, key];
        if (typeof value === "string") {
            const manifestKey = nextPath.join(".");
            if (Object.hasOwn(target, manifestKey)) throw new Error(`Duplicate selector manifest key: ${manifestKey}`);
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
const flattenDynamicSelectors = (tree, prefix = [], target = {}) => {
    for (const [key, value] of Object.entries(tree)) {
        const nextPath = [...prefix, key];
        if (isDynamicDefinition(value)) {
            const manifestKey = nextPath.join(".");
            const paramEntries = Object.entries(value.params ?? {});
            if (Object.hasOwn(target, manifestKey)) throw new Error(`Duplicate selector manifest key: ${manifestKey}`);
            target[manifestKey] = {
                description: value.description,
                selectorPattern: value.selectorPattern ?? (value.testIdPattern ? testIdPatternSelector(value.testIdPattern) : ""),
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
const mergeLiteralAndDynamicNodes = (literalNode, dynamicNode, path = []) => {
    const merged = {};
    const keys = new Set([
        ...Object.keys(literalNode ?? {}),
        ...Object.keys(dynamicNode ?? {}),
    ]);
    keys.forEach((key) => {
        const literalValue = literalNode?.[key];
        const dynamicValue = dynamicNode?.[key];
        const nextPath = [...path, key];
        if (literalValue !== undefined && dynamicValue !== undefined &&
            (typeof literalValue === 'string' || isDynamicDefinition(dynamicValue))) {
            throw new Error(`Selector '${nextPath.join('.')}' has conflicting literal and dynamic definitions`);
        }
        if (typeof literalValue === "string") {
            merged[key] = literalValue;
            return;
        }
        if (literalValue && typeof literalValue === "object") {
            merged[key] = mergeLiteralAndDynamicNodes(literalValue, isDynamicDefinition(dynamicValue) ? undefined : dynamicValue, nextPath);
            return;
        }
        if (dynamicValue) {
            if (isDynamicDefinition(dynamicValue)) {
                merged[key] = createDynamicSelectorFn(dynamicValue, nextPath.join("."));
                return;
            }
            merged[key] = mergeLiteralAndDynamicNodes(undefined, dynamicValue, nextPath);
        }
    });
    return merged;
};
const createDynamicSelectorFn = (definition, path) => {
    return (params) => {
        const normalized = normalizeParams(definition, params ?? {}, path);
        const template = definition.testIdPattern ?? definition.selectorPattern;
        if (!template) {
            throw new Error(`Selector '${path}' is missing both testIdPattern and selectorPattern`);
        }
        return formatTemplate(template, normalized, path, !definition.testIdPattern);
    };
};
export const defineDynamicSelector = (definition) => {
    if (Boolean(definition.testIdPattern) === Boolean(definition.selectorPattern)) {
        throw new Error('Dynamic selector must specify exactly one testIdPattern or selectorPattern');
    }
    const tokens = [...(definition.testIdPattern ?? definition.selectorPattern).matchAll(TEMPLATE_TOKEN)].map(m => m[1]);
    const params = Object.keys(definition.params ?? {});
    if (tokens.some(token => !params.includes(token)) || params.some(param => !tokens.includes(param))) {
        throw new Error('Selector pattern tokens and declared parameters must agree');
    }
    return { ...definition, kind: 'dynamic-selector' };
};
const mergeLibrary = (local, library, path = 'library') => {
    const result = { ...local };
    for (const [key, value] of Object.entries(library)) {
        if (Object.hasOwn(result, key)) {
            if (typeof result[key] !== 'object' || typeof value !== 'object') {
                throw new Error(`Conflicting library selector: ${path}.${key}`);
            }
            result[key] = mergeLibrary(result[key], value, `${path}.${key}`);
        } else result[key] = value;
    }
    return result;
};
export const createSelectorRegistry = (literalTree, dynamicTree, library) => {
    if (library) {
        if (literalTree.library !== undefined && typeof literalTree.library !== 'object') throw new Error('Conflicting library selector namespace');
        literalTree = { ...literalTree, library: mergeLibrary(literalTree.library, library) };
    }
    const selectors = mergeLiteralAndDynamicNodes(literalTree, dynamicTree);
    const manifest = {
        schemaVersion: 1,
        selectors: flattenLiteralSelectors(literalTree),
        dynamicSelectors: flattenDynamicSelectors(dynamicTree),
    };
    const keys = Object.keys(manifest.selectors);
    if (keys.some(key => Object.hasOwn(manifest.dynamicSelectors, key))) throw new Error('Duplicate selector manifest key');
    return { selectors, manifest };
};

// Browser consumers use this accessor; runtime `selectors` preserves raw test IDs.
export function resolveSelector(manifest, key, params = {}) {
    const literal = manifest.selectors?.[key];
    if (literal) {
        if (Object.keys(params).length) throw new Error(`Selector '${key}' received unknown parameters`);
        return literal.selector;
    }
    const entry = manifest.dynamicSelectors?.[key];
    if (!entry) throw new Error(`Unknown selector '${key}'`);
    const definition = { ...entry, params: Object.fromEntries(entry.params.map(p => [p.name, p])) };
    const values = normalizeParams(definition, params, key);
    return entry.testIdPattern
        ? toDataTestIdSelector(formatTemplate(entry.testIdPattern, values, key))
        : formatTemplate(entry.selectorPattern, values, key, true);
}

export function scopeSelector(parent, child) {
    if (!parent?.trim() || !child?.trim()) throw new Error('Both scope and target selectors are required');
    return `:is(${parent}) :is(${child})`;
}
