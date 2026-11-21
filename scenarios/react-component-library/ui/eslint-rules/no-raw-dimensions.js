/**
 * ESLint rule: `design-system/no-raw-dimensions`
 *
 * Rejects raw Tailwind dimension utilities in class strings, so spacing and
 * sizing move through the design ramp rather than being pinned per callsite.
 *
 * Why this rule exists at all: the library corpus is audited by the Go
 * `tokens` catalog gate, but that gate globs `library/**` only. The workspace
 * application under `src/` — the surface a maintainer actually looks at — was
 * covered by nothing, and drifted to 382 raw dimensions across 36 files while
 * the library itself stayed clean. Rather than add a third hand-maintained
 * scope definition in Go, the rule lives here and is applied to both
 * `eslint.config.js` (app) and `eslint.catalog.config.js` (library), which
 * already have the file globs the two surfaces need.
 *
 * Three families, three different fixes. This split is the whole point of the
 * rule; a single "use a semantic token" message is wrong for two of them:
 *
 *   - SPACING (`p-*`, `m-*`, `gap-*`) — the ramp publishes these steps, and
 *     Tailwind's default scale maps onto them exactly (see RAMP_BY_PX). The
 *     fix is mechanical, so it ships as an autofix.
 *
 *   - SIZING (`w-*`, `h-*`) — the ramp deliberately publishes NO size or
 *     icon-size property. Telling an author to "use a declared semantic token"
 *     here sends them looking for something that does not exist. The scale for
 *     icons lives in the Icon primitive's `size` prop, and for boxes the right
 *     answer is usually a layout constraint rather than a fixed dimension.
 *     Both need judgement, so this family reports without an autofix.
 *
 *   - ARBITRARY (`[13px]`) — bypasses the scale entirely. Fixable only when
 *     the value lands exactly on a ramp step; otherwise the honest advice is
 *     that the ramp is missing a rung.
 *
 * Scope is deliberately conservative: only JSX `className`/`class` attributes
 * and arguments to the class-composition helpers below. A broader sweep over
 * every string literal would flag prose containing "p-4" and get the rule
 * disabled, which costs more than the false negatives.
 */

// Ramp steps keyed by the pixel value they resolve to in
// src/design-tokens.css. Tailwind's default spacing scale is retained
// (tailwind.config.ts uses `theme.extend`), so both vocabularies are live and
// a raw step is a choice rather than a necessity.
const RAMP_BY_PX = new Map([
  [4, "space-3xs"],
  [8, "space-2xs"],
  [12, "space-xs"],
  [16, "space-sm"],
  [24, "space-md"],
  [32, "space-lg"],
  [40, "space-xl"],
  [48, "space-2xl"],
]);

// Tailwind's default scale is 0.25rem per step, i.e. step * 4 = pixels.
const TAILWIND_STEP_PX = 4;

const SPACING_PREFIXES = new Set([
  "p", "px", "py", "pt", "pr", "pb", "pl",
  "m", "mx", "my", "mt", "mr", "mb", "ml",
  "gap", "gap-x", "gap-y",
  "space-x", "space-y",
]);

const SIZING_PREFIXES = new Set(["w", "h", "size", "min-w", "min-h", "max-w", "max-h"]);

// Helpers whose string arguments are class lists. Anything else is out of
// scope; see the docblock on why the sweep is narrow.
const CLASS_HELPERS = new Set(["clsx", "cn", "classNames", "classnames", "twMerge", "cva"]);

const CLASS_ATTRIBUTES = new Set(["className", "class"]);

// A dimension token: optional responsive/state prefixes (`md:`, `hover:`),
// then a known property prefix, then either a numeric step or an arbitrary
// bracket value. Negative margins (`-mt-2`) are included.
const TOKEN_RE = /^((?:[a-z0-9-]+:)*)(-?)([a-z]+(?:-[xy])?|min-[wh]|max-[wh])-(\[[^\]]+\]|[0-9]+(?:\.[0-9]+)?)$/;

const PX_ARBITRARY_RE = /^\[([0-9.]+)px\]$/;

/** Classify one whitespace-delimited class token, or null when it is not a dimension. */
function classify(token) {
  const match = TOKEN_RE.exec(token);
  if (!match) return null;
  const [, variants, negative, property, value] = match;
  const isSpacing = SPACING_PREFIXES.has(property);
  const isSizing = SIZING_PREFIXES.has(property);
  if (!isSpacing && !isSizing) return null;

  const arbitrary = PX_ARBITRARY_RE.exec(value);
  if (value.startsWith("[")) {
    const px = arbitrary ? Number(arbitrary[1]) : null;
    return { kind: "arbitrary", token, variants, negative, property, px, rampStep: px === null ? null : RAMP_BY_PX.get(px) ?? null };
  }
  if (isSizing) {
    // `w-0`/`h-0` collapse a box deliberately; there is no size scale to move
    // them onto, so they are not a ramp defect.
    if (Number(value) === 0) return null;
    return { kind: "sizing", token, variants, negative, property, px: null, rampStep: null };
  }
  const px = Number(value) * TAILWIND_STEP_PX;
  // Zero is the absence of spacing, not an untokenized amount of it. The ramp
  // has no zero rung and should not grow one — `p-0` is the correct way to say
  // "no padding", and flagging it would train readers to ignore this rule.
  if (px === 0) return null;
  return { kind: "spacing", token, variants, negative, property, px, rampStep: RAMP_BY_PX.get(px) ?? null };
}

/** Rebuild a token against a ramp step, preserving variants and negation. */
function tokenWithRamp(finding) {
  return `${finding.variants}${finding.negative}${finding.property}-${finding.rampStep}`;
}

/**
 * Report every dimension token inside a class-list string node.
 * `node` must be a Literal or TemplateElement whose raw text we can offset into.
 */
function reportNode(context, node, text, textStartOffset) {
  let cursor = 0;
  for (const raw of text.split(/(\s+)/)) {
    const start = cursor;
    cursor += raw.length;
    const token = raw.trim();
    if (!token) continue;
    const finding = classify(token);
    if (!finding) continue;

    const range = [textStartOffset + start, textStartOffset + start + token.length];
    const replacement = finding.rampStep ? tokenWithRamp(finding) : null;

    if (finding.kind === "spacing") {
      context.report({
        node,
        loc: {
          start: context.sourceCode.getLocFromIndex(range[0]),
          end: context.sourceCode.getLocFromIndex(range[1]),
        },
        messageId: replacement ? "spacingFixable" : "spacingUnmapped",
        data: { token, replacement: replacement ?? "", px: String(finding.px) },
        fix: replacement ? (fixer) => fixer.replaceTextRange(range, replacement) : undefined,
      });
      continue;
    }

    if (finding.kind === "sizing") {
      context.report({
        node,
        loc: {
          start: context.sourceCode.getLocFromIndex(range[0]),
          end: context.sourceCode.getLocFromIndex(range[1]),
        },
        messageId: "sizing",
        data: { token },
      });
      continue;
    }

    context.report({
      node,
      loc: {
        start: context.sourceCode.getLocFromIndex(range[0]),
        end: context.sourceCode.getLocFromIndex(range[1]),
      },
      messageId: replacement ? "arbitraryFixable" : "arbitrary",
      data: { token, replacement: replacement ?? "" },
      fix: replacement ? (fixer) => fixer.replaceTextRange(range, replacement) : undefined,
    });
  }
}

export default {
  meta: {
    type: "problem",
    fixable: "code",
    docs: {
      description:
        "Spacing and sizing must move through the design ramp. Raw Tailwind dimension utilities pin a value at one callsite where the ramp cannot reach it.",
    },
    schema: [],
    messages: {
      spacingFixable:
        "'{{token}}' is a raw {{px}}px spacing step. Use '{{replacement}}' so this element tracks the ramp: a raw step does not move when the ramp is retuned, so it drifts out of rhythm with every tokenized neighbour the first time density changes.",
      spacingUnmapped:
        "'{{token}}' is a raw {{px}}px spacing step that no published ramp utility resolves to. Use the nearest published step (space-3xs 4px through space-2xl 48px). If {{px}}px is genuinely required, add the rung in BOTH places it has to exist: the custom property in src/design-tokens.css and the Tailwind alias in tailwind.theme.json — a property declared without its alias is reachable from CSS but not from a class name, which is why some ramp steps appear to be missing when they are only half-published.",
      sizing:
        "'{{token}}' sizes an element with a raw dimension. For icons use the Icon primitive's size scale (size=\"sm\" | \"md\" | \"lg\") — the ramp deliberately publishes no icon-size property, so there is no token to substitute here. For non-icon boxes prefer a layout constraint (flex/grid) over a fixed dimension.",
      arbitraryFixable:
        "'{{token}}' bypasses the scale but lands exactly on a ramp step. Use '{{replacement}}'.",
      arbitrary:
        "'{{token}}' is an arbitrary value that bypasses the ramp entirely. If the nearest ramp step is visually acceptable, use it; if no step fits, the ramp is missing a rung — add it in src/design-tokens.css rather than encoding the exception at one callsite where nothing can find it later.",
    },
  },
  create(context) {
    /** Walk a node that is expected to hold a class list. */
    const visitClassValue = (value) => {
      if (!value) return;
      if (value.type === "Literal" && typeof value.value === "string") {
        // +1 skips the opening quote.
        reportNode(context, value, value.value, value.range[0] + 1);
        return;
      }
      if (value.type === "JSXExpressionContainer") {
        visitClassValue(value.expression);
        return;
      }
      if (value.type === "TemplateLiteral") {
        for (const quasi of value.quasis) {
          // +1 skips the backtick or the `}` that closes the previous expression.
          reportNode(context, quasi, quasi.value.raw, quasi.range[0] + 1);
        }
        return;
      }
      if (value.type === "ConditionalExpression") {
        visitClassValue(value.consequent);
        visitClassValue(value.alternate);
        return;
      }
      if (value.type === "LogicalExpression") {
        visitClassValue(value.left);
        visitClassValue(value.right);
      }
    };

    return {
      JSXAttribute(node) {
        if (node.name?.type !== "JSXIdentifier" || !CLASS_ATTRIBUTES.has(node.name.name)) return;
        visitClassValue(node.value);
      },
      CallExpression(node) {
        const name =
          node.callee.type === "Identifier"
            ? node.callee.name
            : node.callee.type === "MemberExpression" && node.callee.property.type === "Identifier"
              ? node.callee.property.name
              : null;
        if (!name || !CLASS_HELPERS.has(name)) return;
        for (const argument of node.arguments) visitClassValue(argument);
      },
    };
  },
};
