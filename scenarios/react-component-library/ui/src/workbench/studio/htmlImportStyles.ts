/** Static presentation CSS retained by the inert HTML import pipeline. */

export type HtmlCssRule = { selector: string; declarations: Map<string, string>; order: number };
export type HtmlPseudoRule = {
  selector: string;
  pseudo: "before" | "after";
  declarations: Map<string, string>;
  content?: string;
  order: number;
};

const HTML_STYLE_PROPERTIES = new Set([
  "align-content",
  "align-items",
  "align-self",
  "aspect-ratio",
  "background",
  "background-color",
  "background-image",
  "background-position",
  "background-repeat",
  "background-size",
  "backdrop-filter",
  "border",
  "border-bottom",
  "border-left",
  "border-color",
  "border-radius",
  "border-style",
  "border-top",
  "border-width",
  "box-shadow",
  "box-sizing",
  "color",
  "column-gap",
  "display",
  "flex",
  "flex-basis",
  "flex-direction",
  "flex-grow",
  "flex-shrink",
  "flex-wrap",
  "font-family",
  "font",
  "font-feature-settings",
  "font-size",
  "font-style",
  "font-variant-numeric",
  "font-weight",
  "gap",
  "grid-auto-flow",
  "grid-column",
  "grid-row",
  "grid-template-columns",
  "grid-template-rows",
  "height",
  "justify-content",
  "justify-items",
  "letter-spacing",
  "line-height",
  "list-style",
  "left",
  "margin",
  "margin-bottom",
  "margin-left",
  "margin-right",
  "margin-top",
  "max-height",
  "max-width",
  "min-height",
  "min-width",
  "object-fit",
  "opacity",
  "outline",
  "outline-offset",
  "overflow",
  "overflow-x",
  "overflow-y",
  "padding",
  "padding-bottom",
  "padding-left",
  "padding-right",
  "padding-top",
  "position",
  "perspective",
  "pointer-events",
  "place-items",
  "right",
  "row-gap",
  "text-align",
  "text-rendering",
  "text-decoration",
  "text-transform",
  "top",
  "transform",
  "transform-origin",
  "transition",
  "white-space",
  "width",
  "z-index",
  "bottom",
  "inset",
]);

export function safeHtmlDeclarations(source: string): Map<string, string> {
  const declarations = new Map<string, string>();
  for (const part of source.split(";")) {
    const colon = part.indexOf(":");
    if (colon < 1) continue;
    const name = part.slice(0, colon).trim().toLowerCase();
    const value = part.slice(colon + 1).trim();
    if (
      !(HTML_STYLE_PROPERTIES.has(name) || /^--[\w-]{1,64}$/.test(name)) ||
      !value ||
      value.length > 160 ||
      /url\s*\(|expression\s*\(|javascript:|@import|[<>;{}]/i.test(value)
    )
      continue;
    declarations.set(name, value);
    if (declarations.size >= 48) break;
  }
  return declarations;
}

export function parseHtmlCssRules(stylesheets: string[]): HtmlCssRule[] {
  const rules: HtmlCssRule[] = [];
  let order = 0;
  for (const stylesheet of stylesheets) {
    for (const match of staticCssRules(stylesheet).matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
      const selectorList = (match[1] || "").trim();
      if (selectorList.startsWith("@")) continue;
      const declarations = safeHtmlDeclarations(match[2] || "");
      if (!declarations.size) continue;
      for (const selector of selectorList.split(",")) {
        const cleanSelector = selector.trim();
        if (/::?(before|after)$/i.test(cleanSelector)) continue;
        if (cleanSelector) rules.push({ selector: cleanSelector, declarations, order: order++ });
      }
    }
  }
  return rules;
}

export function parseHtmlPseudoRules(stylesheets: string[]): HtmlPseudoRule[] {
  const rules: HtmlPseudoRule[] = [];
  let order = 0;
  for (const stylesheet of stylesheets) {
    for (const match of staticCssRules(stylesheet).matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
      const selectorList = (match[1] || "").trim();
      if (selectorList.startsWith("@")) continue;
      const source = match[2] || "";
      const contentMatch = source.match(/(?:^|;)\s*content\s*:\s*([^;]+)\s*(?:;|$)/i);
      const declarations = safeHtmlDeclarations(source);
      if (contentMatch) declarations.delete("content");
      for (const selector of selectorList.split(",")) {
        const pseudo = selector.trim().match(/::?(before|after)$/i);
        if (!pseudo) continue;
        const base =
          selector
            .trim()
            .slice(0, pseudo.index ?? 0)
            .trim() || "*";
        const content = contentMatch?.[1]?.trim();
        rules.push({
          selector: base,
          pseudo: (pseudo[1] || "before").toLowerCase() as "before" | "after",
          declarations,
          ...(content ? { content } : {}),
          order: order++,
        });
      }
    }
  }
  return rules;
}

export function staticCssRules(stylesheet: string): string {
  const source = stylesheet.replace(/\/\*[\s\S]*?\*\//g, "");
  let output = "";
  for (let index = 0; index < source.length; ) {
    const atRule = source.slice(index).match(/^\s*@[-\w]+[^{}]*\{/);
    if (atRule) {
      let cursor = index + atRule[0].length;
      let depth = 1;
      for (; cursor < source.length && depth > 0; cursor += 1) {
        if (source[cursor] === "{") depth += 1;
        else if (source[cursor] === "}") depth -= 1;
      }
      output += " ";
      index = cursor;
      continue;
    }
    output += source.charAt(index);
    index += 1;
  }
  return output;
}

export function htmlStyle(element: Element, rules: HtmlCssRule[]): string | undefined {
  const declarations = new Map<string, string>();
  const matched: HtmlCssRule[] = [];
  for (const rule of rules) {
    try {
      if (element.matches(rule.selector)) matched.push(rule);
    } catch {
      /* Ignore selectors unsupported by the detached DOM. */
    }
  }
  matched.sort(compareCssSpecificity);
  for (const rule of matched) {
    for (const [name, value] of rule.declarations) declarations.set(name, value);
  }
  for (const [name, value] of safeHtmlDeclarations(element.getAttribute("style") || "")) {
    declarations.set(name, value);
  }
  return declarations.size ? JSON.stringify(Object.fromEntries(declarations)) : undefined;
}

export function compareCssSpecificity(
  left: { selector: string; order: number },
  right: { selector: string; order: number },
): number {
  const specificity = (selector: string): [number, number, number] => {
    const clean = selector.replace(/::[\w-]+/g, "");
    return [
      (clean.match(/#[\w-]+/g) || []).length,
      (clean.match(/\.[\w-]+|\[[^\]]+\]|:(?!:)[\w-]+(?:\([^)]*\))?/g) || []).length,
      (clean.match(/(?:^|[\s>+~])(?:[a-z][\w-]*)/gi) || []).length,
    ];
  };
  const a = specificity(left.selector);
  const b = specificity(right.selector);
  return a[0] - b[0] || a[1] - b[1] || a[2] - b[2] || left.order - right.order;
}

export function hasVisualOrGeometryStyle(serialized: string | undefined): boolean {
  if (!serialized) return false;
  try {
    const style = JSON.parse(serialized) as Record<string, string>;
    return Object.keys(style).some((property) =>
      [
        "background",
        "background-color",
        "background-image",
        "border",
        "border-bottom",
        "border-color",
        "border-left",
        "border-radius",
        "border-top",
        "box-shadow",
        "height",
        "outline",
        "width",
      ].includes(property),
    );
  } catch {
    return false;
  }
}

export function ensureInheritedTextColor(serialized: string | undefined): string {
  try {
    const declarations = serialized ? (JSON.parse(serialized) as Record<string, string>) : {};
    if (!declarations.color) declarations.color = "inherit";
    return JSON.stringify(declarations);
  } catch {
    return serialized || '{"color":"inherit"}';
  }
}

