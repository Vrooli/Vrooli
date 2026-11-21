export type StrictTokenDiagnostic = {
  schema: 'bas-strict-token-diagnostic/v1';
  token_count: number;
  sentinel_element_count: number;
  non_sentinel_color_count: number;
};

/** CSSOM based token sentinel used by UI Health's strict-token diagnostic. */
export function applyStrictTokenSentinelInPage(): StrictTokenDiagnostic {
  const names = new Set<string>();
  const visit = (rules: CSSRuleList | undefined): void => {
    for (const rule of Array.from(rules || [])) {
      if (rule instanceof CSSStyleRule)
        for (let index = 0; index < rule.style.length; index++) {
          const name = rule.style.item(index);
          if (name.startsWith('--')) names.add(name);
        }
      const nested = (rule as CSSGroupingRule).cssRules;
      if (nested) visit(nested);
    }
  };
  for (const sheet of Array.from(document.styleSheets)) {
    try {
      visit(sheet.cssRules);
    } catch {
      // Cross-origin stylesheets remain outside the inspected token set.
    }
  }
  for (const style of Array.from(document.querySelectorAll('style'))) {
    for (const [, name] of (style.textContent || '').matchAll(/(--[A-Za-z0-9_-]+)\s*:/g))
      if (name) names.add(name);
  }
  for (const name of names) document.documentElement.style.setProperty(name, '#ff00ff');

  let sentinel = 0;
  let nonSentinel = 0;
  for (const element of Array.from(document.querySelectorAll('body *'))) {
    const style = getComputedStyle(element);
    const color = style.color.replace(/\s/g, '').toLowerCase();
    const background = style.backgroundColor.replace(/\s/g, '').toLowerCase();
    const hit =
      color === 'rgb(255,0,255)' ||
      background === 'rgb(255,0,255)' ||
      color === '#ff00ff' ||
      background === '#ff00ff';
    if (hit) sentinel++;
    else if (color && color !== 'rgba(0,0,0,0)' && background !== 'rgba(0,0,0,0)') nonSentinel++;
  }
  return {
    schema: 'bas-strict-token-diagnostic/v1',
    token_count: names.size,
    sentinel_element_count: sentinel,
    non_sentinel_color_count: nonSentinel,
  };
}
