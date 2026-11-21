/** Injectable L3 style census. Keep collectStyleCensusInPage closure-free so
 * BAS can serialize it into Page.evaluate(). */

export type StyleCategory =
  | 'color'
  | 'typography'
  | 'spacing'
  | 'layout'
  | 'radius'
  | 'shadow'
  | 'effects'
  | 'variant';
export type StyleProperty =
  | 'foreground'
  | 'background'
  | 'border-top'
  | 'outline'
  | 'font-family'
  | 'font-size'
  | 'font-weight'
  | 'line-height'
  | 'letter-spacing'
  | 'margin-top'
  | 'margin-right'
  | 'margin-bottom'
  | 'margin-left'
  | 'padding-top'
  | 'padding-right'
  | 'padding-bottom'
  | 'padding-left'
  | 'gap'
  | 'row-gap'
  | 'column-gap'
  | 'width'
  | 'min-width'
  | 'max-width'
  | 'height'
  | 'min-height'
  | 'max-height'
  | 'border-top-width'
  | 'outline-width'
  | 'outline-offset'
  | 'opacity'
  | 'z-index'
  | 'transition-duration'
  | 'transition-timing-function'
  | 'radius-top-left'
  | 'radius-top-right'
  | 'radius-bottom-right'
  | 'radius-bottom-left'
  | 'box-shadow'
  | 'variant';
export type StyleToken = {
  name: string;
  category: StyleCategory;
  property: StyleProperty;
  value: string;
};
export type StyleCensusOptions = { tokens?: StyleToken[]; maxElements?: number };
export type StyleSource = { selector: string; tag: string; className: string };
export type StyleCluster = {
  id: string;
  category: StyleCategory;
  property: StyleProperty;
  value: string;
  count: number;
  sources: StyleSource[];
  token: { name: string; property: StyleProperty; value: string; distance: number } | null;
  tokenMissingReason?: 'catalog-not-provided' | 'no-comparable-token';
};
export type StyleCensusResult = {
  schema: 'bas-agent-style-census/v1';
  tokenCatalogStatus: 'rcl' | 'unavailable';
  elementCount: number;
  sampledElementCount: number;
  clusters: StyleCluster[];
};

export function collectStyleCensusInPage(options: StyleCensusOptions = {}): StyleCensusResult {
  const categories: StyleCategory[] = [
    'color',
    'typography',
    'spacing',
    'layout',
    'radius',
    'shadow',
    'effects',
    'variant',
  ];
  const tokens =
    Array.isArray(options.tokens) && options.tokens.length ? options.tokens : undefined;
  const maxElements = Math.max(1, Math.floor(options.maxElements ?? 5000));
  const elements = Array.from(document.querySelectorAll('body *'))
    .filter((element) => {
      const style = getComputedStyle(element);
      const rect = element.getBoundingClientRect();
      return (
        style.display !== 'none' &&
        style.visibility !== 'hidden' &&
        Number(style.opacity) !== 0 &&
        rect.width > 0 &&
        rect.height > 0
      );
    })
    .slice(0, maxElements);
  const collected = new Map<
    string,
    {
      category: StyleCategory;
      property: StyleProperty;
      value: string;
      count: number;
      sources: StyleSource[];
    }
  >();
  const clean = (value: string): string => value.trim().replace(/\s+/g, ' ');
  const sourceOf = (element: Element): StyleSource => {
    const id = element.getAttribute('id');
    const testId = element.getAttribute('data-testid');
    const classes =
      typeof (element as HTMLElement).className === 'string'
        ? clean((element as HTMLElement).className)
        : '';
    const selector = id
      ? `#${id}`
      : testId
        ? `[data-testid="${testId}"]`
        : `${element.tagName.toLowerCase()}${classes ? `.${classes.split(' ').slice(0, 3).join('.')}` : ''}`;
    return { selector, tag: element.tagName.toLowerCase(), className: classes };
  };
  const add = (
    category: StyleCategory,
    property: StyleProperty,
    raw: string,
    source: StyleSource
  ): void => {
    const value = clean(raw);
    if (!value || value === 'none' || value === 'normal' || value === 'auto' || value === '0px')
      return;
    const key = `${category}\u0000${property}\u0000${value}`;
    const current = collected.get(key) ?? { category, property, value, count: 0, sources: [] };
    current.count++;
    if (
      !current.sources.some((item) => item.selector === source.selector) &&
      current.sources.length < 12
    )
      current.sources.push(source);
    collected.set(key, current);
  };
  const parse = (category: StyleCategory, value: string): number[] | null => {
    if (category === 'color') {
      const hex = value.match(/^#([\da-f]{3,8})$/i)?.[1];
      if (hex) {
        const expanded =
          hex.length === 3 || hex.length === 4
            ? hex
                .split('')
                .map((c) => c + c)
                .join('')
            : hex;
        if (expanded.length === 6 || expanded.length === 8)
          return [0, 2, 4].map((i) => parseInt(expanded.slice(i, i + 2), 16));
      }
      const rgb = value.match(/rgba?\(\s*([\d.]+)[, ]+\s*([\d.]+)[, ]+\s*([\d.]+)/i);
      return rgb ? [Number(rgb[1]), Number(rgb[2]), Number(rgb[3])] : null;
    }
    if (
      category === 'spacing' ||
      category === 'layout' ||
      category === 'effects' ||
      category === 'radius' ||
      category === 'typography'
    ) {
      const nums = value.match(/-?\d+(?:\.\d+)?/g)?.map(Number);
      return nums?.length ? nums : null;
    }
    return null;
  };
  for (const element of elements) {
    const style = getComputedStyle(element);
    const source = sourceOf(element);
    add('color', 'foreground', style.color, source);
    add('color', 'background', style.backgroundColor, source);
    add('color', 'border-top', style.borderTopColor, source);
    add('color', 'outline', style.outlineColor, source);
    add('typography', 'font-family', style.fontFamily, source);
    add('typography', 'font-size', style.fontSize, source);
    add('typography', 'font-weight', style.fontWeight, source);
    add('typography', 'line-height', style.lineHeight, source);
    add('typography', 'letter-spacing', style.letterSpacing, source);
    add('spacing', 'margin-top', style.marginTop, source);
    add('spacing', 'margin-right', style.marginRight, source);
    add('spacing', 'margin-bottom', style.marginBottom, source);
    add('spacing', 'margin-left', style.marginLeft, source);
    add('spacing', 'padding-top', style.paddingTop, source);
    add('spacing', 'padding-right', style.paddingRight, source);
    add('spacing', 'padding-bottom', style.paddingBottom, source);
    add('spacing', 'padding-left', style.paddingLeft, source);
    add('spacing', 'gap', style.gap, source);
    add('spacing', 'row-gap', style.rowGap, source);
    add('spacing', 'column-gap', style.columnGap, source);
    add('layout', 'width', style.width, source);
    add('layout', 'min-width', style.minWidth, source);
    add('layout', 'max-width', style.maxWidth, source);
    add('layout', 'height', style.height, source);
    add('layout', 'min-height', style.minHeight, source);
    add('layout', 'max-height', style.maxHeight, source);
    add('layout', 'border-top-width', style.borderTopWidth, source);
    add('layout', 'outline-width', style.outlineWidth, source);
    add('layout', 'outline-offset', style.outlineOffset, source);
    add('radius', 'radius-top-left', style.borderTopLeftRadius, source);
    add('radius', 'radius-top-right', style.borderTopRightRadius, source);
    add('radius', 'radius-bottom-right', style.borderBottomRightRadius, source);
    add('radius', 'radius-bottom-left', style.borderBottomLeftRadius, source);
    add('shadow', 'box-shadow', style.boxShadow, source);
    add('effects', 'opacity', style.opacity, source);
    add('effects', 'z-index', style.zIndex, source);
    add('effects', 'transition-duration', style.transitionDuration, source);
    add('effects', 'transition-timing-function', style.transitionTimingFunction, source);
    const variant = [
      element.getAttribute('data-variant'),
      element.getAttribute('aria-pressed') === 'true' ? 'pressed' : '',
      element.getAttribute('aria-selected') === 'true' ? 'selected' : '',
    ]
      .filter(Boolean)
      .join(' ');
    add('variant', 'variant', variant, source);
  }
  const clusters: StyleCluster[] = Array.from(collected.values())
    .map((item) => {
      const candidates =
        tokens?.filter(
          (token) => token.category === item.category && token.property === item.property
        ) ?? [];
      let best: { name: string; property: StyleProperty; value: string; distance: number } | null =
        null;
      for (const token of candidates) {
        let distance: number | null =
          item.value.toLowerCase() === token.value.trim().toLowerCase() ? 0 : null;
        if (distance === null) {
          const left = parse(item.category, item.value);
          const right = parse(item.category, token.value);
          if (left && right && left.length === right.length)
            distance = Math.sqrt(
              left.reduce((sum, n, index) => sum + (n - (right[index] ?? n)) ** 2, 0)
            );
        }
        if (
          distance !== null &&
          (!best ||
            distance < best.distance ||
            (distance === best.distance && token.name.localeCompare(best.name) < 0))
        )
          best = {
            name: token.name,
            property: token.property,
            value: token.value,
            distance: Math.round(distance * 1000) / 1000,
          };
      }
      return {
        id: '',
        ...item,
        token: best,
        ...(!best
          ? {
              tokenMissingReason: tokens
                ? ('no-comparable-token' as const)
                : ('catalog-not-provided' as const),
            }
          : {}),
      };
    })
    .sort(
      (a, b) =>
        categories.indexOf(a.category) - categories.indexOf(b.category) ||
        a.property.localeCompare(b.property) ||
        a.value.localeCompare(b.value)
    );
  // IDs are assigned after sorting so serialization is deterministic.
  let prior = '';
  let index = 0;
  for (const cluster of clusters) {
    if (cluster.category !== prior) {
      prior = cluster.category;
      index = 0;
    }
    cluster.id = `${cluster.category}-${++index}`;
  }
  return {
    schema: 'bas-agent-style-census/v1',
    tokenCatalogStatus: tokens ? 'rcl' : 'unavailable',
    elementCount: document.querySelectorAll('body *').length,
    sampledElementCount: elements.length,
    clusters,
  };
}

/** Uses the page's loaded RCL token layer when present; arbitrary CSS variables are not mislabeled as RCL tokens. */
export function collectRclStyleTokensInPage(): StyleToken[] {
  const computed = getComputedStyle(document.documentElement);
  const tokens: StyleToken[] = [];
  const probe = document.createElement?.('div') as HTMLElement | undefined;
  if (probe?.style && document.body) {
    probe.style.cssText =
      'position:fixed;visibility:hidden;contain:strict;left:-10000px;top:-10000px';
    document.body.appendChild(probe);
  }
  const spacingProperties: StyleProperty[] = [
    'margin-top',
    'margin-right',
    'margin-bottom',
    'margin-left',
    'padding-top',
    'padding-right',
    'padding-bottom',
    'padding-left',
    'gap',
    'row-gap',
    'column-gap',
  ];
  const radiusProperties: StyleProperty[] = [
    'radius-top-left',
    'radius-top-right',
    'radius-bottom-right',
    'radius-bottom-left',
  ];
  const colorProperties: StyleProperty[] = ['foreground', 'background', 'border-top', 'outline'];
  const cssProperty: Record<StyleProperty, string> = {
    foreground: 'color',
    background: 'background-color',
    'border-top': 'border-top-color',
    outline: 'outline-color',
    'font-family': 'font-family',
    'font-size': 'font-size',
    'font-weight': 'font-weight',
    'line-height': 'line-height',
    'letter-spacing': 'letter-spacing',
    'margin-top': 'margin-top',
    'margin-right': 'margin-right',
    'margin-bottom': 'margin-bottom',
    'margin-left': 'margin-left',
    'padding-top': 'padding-top',
    'padding-right': 'padding-right',
    'padding-bottom': 'padding-bottom',
    'padding-left': 'padding-left',
    gap: 'gap',
    'row-gap': 'row-gap',
    'column-gap': 'column-gap',
    width: 'width',
    'min-width': 'min-width',
    'max-width': 'max-width',
    height: 'height',
    'min-height': 'min-height',
    'max-height': 'max-height',
    'border-top-width': 'border-top-width',
    'outline-width': 'outline-width',
    'outline-offset': 'outline-offset',
    'radius-top-left': 'border-top-left-radius',
    'radius-top-right': 'border-top-right-radius',
    'radius-bottom-right': 'border-bottom-right-radius',
    'radius-bottom-left': 'border-bottom-left-radius',
    'box-shadow': 'box-shadow',
    opacity: 'opacity',
    'z-index': 'z-index',
    'transition-duration': 'transition-duration',
    'transition-timing-function': 'transition-timing-function',
    variant: 'visibility',
  };
  const resolvedValue = (name: string, property: StyleProperty, fallback: string): string => {
    const target = cssProperty[property];
    if (!probe?.style || !document.body || !target) return fallback;
    probe.style.removeProperty(target);
    probe.style.setProperty(target, `var(${name})`);
    const value = getComputedStyle(probe).getPropertyValue(target).trim();
    return value || fallback;
  };
  const add = (
    name: string,
    category: StyleCategory,
    properties: StyleProperty[],
    value: string,
    sourceProperty?: StyleProperty
  ): void => {
    if (!value) return;
    for (const property of properties) {
      tokens.push({
        name,
        category,
        property,
        value: resolvedValue(name, sourceProperty ?? property, value),
      });
    }
  };
  // The app shell may project the RCL token contract into its own :root rule
  // instead of shipping RCL's @layer stylesheet verbatim. Recognize only the
  // published token families; arbitrary application variables stay excluded.
  for (let index = 0; index < computed.length; index++) {
    const name = computed.item(index);
    if (!name?.startsWith('--')) continue;
    const value = computed.getPropertyValue(name).trim();
    if (
      /^--(?:color-|app-)/.test(name) ||
      /^--(?:markdown-(?:border|code-surface|code-text|error|link|muted)|scrollbar-thumb(?:-hover)?|provenance-[\w-]+|badge-border|surface-border|spinner-border|focus-ring-color|edge-fade)$/.test(
        name
      )
    ) {
      const semantic = name.toLowerCase();
      const properties: StyleProperty[] = [];
      if (/(?:foreground|text|link|error)(?:-|$)/.test(semantic)) properties.push('foreground');
      if (/(?:background|surface|shell|overlay|field|subtle|scrim|edge-fade)$/.test(semantic))
        properties.push('background');
      if (/border/.test(semantic)) properties.push('border-top');
      if (/(?:focus|ring)(?:-|$)/.test(semantic)) properties.push('outline');
      if (/^--color-(?:accent|danger|info|primary|success|warning)(?:-|$)/.test(semantic))
        properties.push('foreground', 'background');
      if (/^--color-on-/.test(semantic)) properties.push('foreground');
      add(name, 'color', properties.length ? properties : colorProperties, value);
    } else if (/^--(?:font-family|font)-(?:sans|mono)$/.test(name))
      add(name, 'typography', ['font-family'], value);
    else if (/^--(?:font-size-|text-.+-size$|text-(?:xs|wall)$)/.test(name))
      add(name, 'typography', ['font-size'], value);
    else if (/^--(?:text-.+-weight|weight-)/.test(name))
      add(name, 'typography', ['font-weight'], value);
    else if (/^--(?:text-.+-line|skeleton-line-height$)/.test(name))
      add(name, 'typography', ['line-height'], value);
    else if (/^--(?:text-.+-tracking|tracking-)/.test(name))
      add(name, 'typography', ['letter-spacing'], value);
    else if (
      /^--text-(?:body|body-sm|caption|code|display|heading|heading-(?:lg|sm)|label|overline|subtitle|title)$/.test(
        name
      )
    ) {
      if (probe?.style && document.body) {
        probe.style.removeProperty('font');
        probe.style.setProperty('font', `var(${name})`);
        const style = getComputedStyle(probe);
        for (const property of [
          'font-family',
          'font-size',
          'font-weight',
          'line-height',
        ] as const) {
          const resolved = style.getPropertyValue(cssProperty[property]).trim();
          if (resolved) tokens.push({ name, category: 'typography', property, value: resolved });
        }
      } else {
        for (const property of ['font-family', 'font-size', 'font-weight', 'line-height'] as const)
          add(name, 'typography', [property], value);
      }
    } else if (/^--(?:space-|spacing-)/.test(name)) add(name, 'spacing', spacingProperties, value);
    else if (/^--(?:panel-padding|control-padding)$/.test(name))
      add(
        name,
        'spacing',
        spacingProperties.filter((property) => property.startsWith('padding-')),
        value
      );
    else if (
      /^--(?:control-height(?:-.+)?|control-size-.+|icon-size-.+|tap-target-min|touch-target|content-min-height|sidebar-(?:min-|max-)?width|container-(?:content|wide)|overlay-dialog-.+|overlay-grabber-(?:block|inline)|edge-fade-width)$/.test(
        name
      )
    ) {
      const properties: StyleProperty[] = [];
      if (/sidebar-(?:min-|max-)?width|container-|overlay-dialog-/.test(name)) {
        properties.push(
          name.includes('sidebar-min')
            ? 'min-width'
            : name.includes('sidebar-max') ||
                name.includes('container-') ||
                name.includes('overlay-dialog-')
              ? 'max-width'
              : 'width'
        );
      } else if (name === '--content-min-height') properties.push('min-height');
      else if (name === '--edge-fade-width') properties.push('width');
      else if (name === '--overlay-grabber-inline') properties.push('width');
      else if (name === '--overlay-grabber-block') properties.push('height');
      else if (/control-size-|icon-size-|tap-target-min|touch-target/.test(name))
        properties.push('width', 'min-width', 'height', 'min-height');
      else properties.push('height', 'min-height');
      add(name, 'layout', properties, value);
    } else if (/^--overlay-drawer-top-gap$/.test(name)) add(name, 'spacing', ['margin-top'], value);
    else if (/^--overlay-menu-align$/.test(name)) add(name, 'spacing', ['margin-left'], value);
    else if (/^--radius-/.test(name)) add(name, 'radius', radiusProperties, value);
    else if (/^--(?:control-radius|panel-radius)$/.test(name))
      add(name, 'radius', radiusProperties, value);
    else if (/^--(?:elev-|shadow-|focus-ring$|glow-primary$)/.test(name))
      add(name, 'shadow', ['box-shadow'], value);
    else if (/^--(?:opacity-)/.test(name)) add(name, 'effects', ['opacity'], value);
    else if (/^--layer-/.test(name)) add(name, 'effects', ['z-index'], value);
    else if (/^--dur-|^--motion-/.test(name)) add(name, 'effects', ['transition-duration'], value);
    else if (/^--(?:ease-|spring-)/.test(name))
      add(name, 'effects', ['transition-timing-function'], value);
    else if (/^--(?:border-|separator-thickness$)/.test(name))
      add(name, 'layout', ['border-top-width'], value);
    else if (/^--focus-ring-width$/.test(name)) add(name, 'layout', ['outline-width'], value);
    else if (/^--focus-ring-offset$/.test(name)) add(name, 'layout', ['outline-offset'], value);
    else if (/^--control-border$/.test(name)) {
      let width = value.match(/^\s*([\d.]+(?:px|rem|em))\s+/i)?.[1] ?? '';
      if (probe?.style && document.body) {
        probe.style.removeProperty('border-top');
        probe.style.setProperty('border-top', `var(${name})`);
        width = getComputedStyle(probe).getPropertyValue('border-top-width').trim() || width;
      }
      tokens.push({
        name,
        category: 'layout',
        property: 'border-top-width',
        value: width || value,
      });
    }
  }
  probe?.remove();
  return tokens;
}

export function captureStyleCensusInPage(options: StyleCensusOptions = {}): StyleCensusResult {
  return collectStyleCensusInPage(options);
}
