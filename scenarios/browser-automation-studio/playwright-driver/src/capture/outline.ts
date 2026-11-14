/** Injectable L2 page outline. The exported collector is self-contained so it
 * can be passed directly to Page.evaluate() by BAS and EX12 consumers. */

export type OutlineRect = { x: number; y: number; width: number; height: number };
export type OutlineNode = {
  id: string;
  kind: 'landmark' | 'region' | 'heading' | 'text' | 'action' | 'repeated-template';
  tag: string;
  role: string | null;
  name: string;
  text?: string;
  level?: number;
  rect: OutlineRect;
  selector: string;
  disabled?: boolean;
  actionType?: string;
  examples?: Array<{ id: string; name: string; text: string; rect: OutlineRect }>;
  actionIds?: string[];
};
export type OutlineResult = {
  schema: 'bas-agent-outline/v1';
  nodes: OutlineNode[];
  unsupported: Array<{ surface: string; reason: string }>;
  tokenBudget: number;
  estimatedTokens: number;
  overBudget: boolean;
  actionCount: number;
};

export type OutlineOptions = { tokenBudget?: number; maxTextLength?: number };

/** Keep this function free of module closures: it is serialized into the page. */
export function collectOutlineInPage(options: OutlineOptions = {}): OutlineResult {
  const doc = document;
  const budget = Math.max(1, Math.floor(options.tokenBudget ?? 8000));
  const maxText = Math.max(40, Math.floor(options.maxTextLength ?? 240));
  const nodes: OutlineNode[] = [];
  const unsupported: Array<{ surface: string; reason: string }> = [];
  let serial = 0;
  const clean = (value: string | null | undefined): string =>
    (value || '').replace(/\s+/g, ' ').trim();
  const visible = (element: Element): boolean => {
    const style = getComputedStyle(element);
    if (style.display === 'none' || style.visibility === 'hidden' || Number(style.opacity) === 0)
      return false;
    const rect = element.getBoundingClientRect();
    return rect.width > 0 && rect.height > 0;
  };
  const rectOf = (element: Element): OutlineRect => {
    const r = element.getBoundingClientRect();
    return {
      x: Math.round(r.x),
      y: Math.round(r.y),
      width: Math.round(r.width),
      height: Math.round(r.height),
    };
  };
  const selectorOf = (element: Element): string => {
    const id = element.getAttribute('id');
    if (id) return `#${CSS.escape(id)}`;
    const testId = element.getAttribute('data-testid');
    if (testId) return `[data-testid="${CSS.escape(testId)}"]`;
    const parts: string[] = [];
    let current: Element | null = element;
    while (current && current !== doc.body && parts.length < 5) {
      const tag = current.tagName.toLowerCase();
      const parent: Element | null = current.parentElement;
      if (!parent) {
        parts.unshift(tag);
        break;
      }
      const siblings = Array.from(parent.children).filter(
        (sibling) => sibling.tagName === current?.tagName
      );
      const index = siblings.indexOf(current) + 1;
      parts.unshift(`${tag}:nth-of-type(${index})`);
      current = parent;
    }
    return parts.join(' > ');
  };
  const nameOf = (element: Element): string =>
    clean(
      element.getAttribute('aria-label') ||
        element.getAttribute('title') ||
        element.getAttribute('alt') ||
        ''
    );
  const isActionElement = (element: Element): boolean => {
    const tag = element.tagName.toLowerCase();
    const role = (element.getAttribute('role') || '').toLowerCase();
    return (
      /^(button|a|input|select|textarea|summary)$/.test(tag) ||
      [
        'button',
        'link',
        'checkbox',
        'radio',
        'switch',
        'tab',
        'menuitem',
        'menuitemcheckbox',
        'menuitemradio',
        'option',
        'combobox',
        'listbox',
        'slider',
        'spinbutton',
        'textbox',
        'searchbox',
        'treeitem',
        'gridcell',
      ].includes(role) ||
      element.hasAttribute('onclick') ||
      element.hasAttribute('tabindex') ||
      ['aria-haspopup', 'aria-expanded', 'aria-pressed', 'aria-selected'].some((attribute) =>
        element.hasAttribute(attribute)
      )
    );
  };
  const textOf = (element: Element): string => clean(element.textContent).slice(0, maxText);
  const add = (
    element: Element,
    kind: OutlineNode['kind'],
    extra: Partial<OutlineNode> = {}
  ): OutlineNode => {
    const node: OutlineNode = {
      id: `n${++serial}`,
      kind,
      tag: element.tagName.toLowerCase(),
      role: element.getAttribute('role'),
      name: nameOf(element),
      text: textOf(element),
      rect: rectOf(element),
      selector: selectorOf(element),
      ...extra,
    };
    nodes.push(node);
    return node;
  };

  const landmarkTags = new Set(['main', 'nav', 'header', 'footer', 'aside', 'form']);
  for (const element of Array.from(doc.querySelectorAll('body *'))) {
    if (!visible(element)) continue;
    const tag = element.tagName.toLowerCase();
    const role = (element.getAttribute('role') || '').toLowerCase();
    if (
      landmarkTags.has(tag) ||
      ['main', 'navigation', 'banner', 'contentinfo', 'complementary', 'search', 'form'].includes(
        role
      )
    )
      add(element, 'landmark');
    if (
      element.hasAttribute('aria-label') &&
      (tag === 'section' || tag === 'article' || role === 'region')
    )
      add(element, 'region');
    if (/^h[1-6]$/.test(tag) || role === 'heading')
      add(element, 'heading', {
        level: Number(tag.slice(1)) || Number(element.getAttribute('aria-level')) || undefined,
      });
    if (isActionElement(element)) {
      const actionNode = add(element, 'action', {
        actionType:
          tag === 'a' || role === 'link'
            ? 'navigate'
            : tag === 'input' || tag === 'textarea' || tag === 'select'
              ? 'input'
              : 'activate',
        disabled:
          (element as HTMLButtonElement).disabled === true ||
          element.getAttribute('aria-disabled') === 'true',
      });
      // Action identity is assigned independently and is never removed by budgeting.
      void actionNode;
    } else if (!element.children.length && textOf(element)) add(element, 'text');
  }

  // Collapse repeated element siblings into one template while keeping each
  // example and each action node independently addressable.
  const groups = new Map<string, Element[]>();
  for (const parent of Array.from(doc.querySelectorAll('body *'))) {
    const buckets = new Map<string, Element[]>();
    for (const child of Array.from(parent.children)) {
      if (!visible(child)) continue;
      const signature = [
        child.tagName,
        child.getAttribute('role') || '',
        child.getAttribute('data-testid') || '',
        child.className && typeof child.className === 'string'
          ? child.className.trim().split(/\s+/).sort().join('.')
          : '',
      ].join('|');
      const bucket = buckets.get(signature) || [];
      bucket.push(child);
      buckets.set(signature, bucket);
    }
    for (const [signature, siblings] of buckets)
      if (siblings.length >= 3) groups.set(`${selectorOf(parent)}|${signature}`, siblings);
  }
  for (const siblings of groups.values()) {
    const sample = siblings[0];
    if (!sample) continue;
    const examples = siblings.slice(0, 3).map((element) => ({
      id: `n${++serial}`,
      name: nameOf(element),
      text: textOf(element),
      rect: rectOf(element),
    }));
    const actionIds = siblings
      .flatMap((element) => {
        return [
          ...(isActionElement(element) ? [element] : []),
          ...Array.from(
            element.querySelectorAll(
              'button,a,input,select,textarea,summary,[role="button"],[role="link"],[role="checkbox"],[role="radio"],[role="switch"],[role="tab"],[role="menuitem"],[role="menuitemcheckbox"],[role="menuitemradio"],[role="option"],[role="combobox"],[role="listbox"],[role="slider"],[role="spinbutton"],[role="textbox"],[role="searchbox"],[role="treeitem"],[role="gridcell"],[tabindex],[onclick],[aria-haspopup],[aria-expanded],[aria-pressed],[aria-selected]'
            )
          ),
        ];
      })
      .map(
        (element) =>
          nodes.find((node) => node.kind === 'action' && node.selector === selectorOf(element))?.id
      )
      .filter((id): id is string => Boolean(id));
    const memberSelectors = new Set(siblings.map(selectorOf));
    for (let index = nodes.length - 1; index >= 0; index--) {
      const node = nodes[index];
      if (node?.selector && memberSelectors.has(node.selector) && node.kind !== 'action')
        nodes.splice(index, 1);
    }
    nodes.push({
      id: `n${++serial}`,
      kind: 'repeated-template',
      tag: sample.tagName.toLowerCase(),
      role: sample.getAttribute('role'),
      name: nameOf(sample),
      text: textOf(sample),
      rect: rectOf(sample),
      selector: selectorOf(sample),
      examples,
      actionIds,
    });
  }

  const addUnsupported = (surface: string, reason: string): void => {
    if (!unsupported.some((item) => item.surface === surface))
      unsupported.push({ surface, reason });
  };
  if (doc.querySelector('iframe'))
    addUnsupported(
      'frames',
      'Embedded frames are not traversed; cross-origin content may be inaccessible.'
    );
  if (doc.querySelector('canvas'))
    addUnsupported(
      'canvas',
      'Canvas pixels and interactive drawing semantics are not represented in the outline.'
    );
  if (doc.querySelector('video,audio'))
    addUnsupported('media', 'Time-based media state is not captured by the outline.');
  if (
    Array.from(doc.querySelectorAll('*')).some((element) =>
      Boolean((element as HTMLElement).shadowRoot)
    )
  )
    addUnsupported('shadow-dom', 'Shadow-root contents are outside the light-DOM outline.');
  if (
    Array.from(doc.querySelectorAll('*')).some((element) => {
      const style = getComputedStyle(element);
      return style.animationName !== 'none' || style.transitionDuration !== '0s';
    })
  )
    addUnsupported(
      'animation',
      'Animated or transitioning surfaces may differ from this single observation.'
    );
  if (doc.querySelector('[aria-rowcount], [aria-setsize], [data-virtualized="true"]'))
    addUnsupported(
      'virtualized-content',
      'Virtualized lists may expose only currently mounted items.'
    );

  // JSON punctuation and identifiers tokenise less compactly than prose.
  // Three UTF-8 bytes per estimated token is a conservative budget signal.
  const estimate = (items: OutlineNode[]): number =>
    Math.ceil(new TextEncoder().encode(JSON.stringify(items)).byteLength / 3);
  const actionCount = nodes.filter((node) => node.kind === 'action').length;
  let estimatedTokens = estimate(nodes);
  if (estimatedTokens > budget) {
    // Reduce non-action copy first; never remove action nodes or action examples.
    for (const node of nodes)
      if (node.kind !== 'action' && node.kind !== 'repeated-template') {
        node.text = '';
        node.name = node.name.slice(0, 48);
      }
    estimatedTokens = estimate(nodes);
  }
  return {
    schema: 'bas-agent-outline/v1',
    nodes,
    unsupported,
    tokenBudget: budget,
    estimatedTokens,
    overBudget: estimatedTokens > budget,
    actionCount,
  };
}

export function captureOutlineInPage(options: OutlineOptions = {}): OutlineResult {
  return collectOutlineInPage(options);
}
