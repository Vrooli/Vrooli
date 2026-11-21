/** L4 exact component identity collected after the privacy prepass. */
export type ComponentIdentity = {
  selector: string;
  asset: string;
  version: string;
  stamp: string;
  source_slot?: string;
};

export type ComponentIdentityResult = {
  schema: 'bas-component-identity/v1';
  element_count: number;
  stamped_count: number;
  components: ComponentIdentity[];
};

/** Closure-free so it can be injected directly into Page.evaluate(). */
export function collectComponentIdentityInPage(): ComponentIdentityResult {
  const selectorOf = (element: Element): string => {
    const id = element.getAttribute('id');
    if (id) return `#${CSS.escape(id)}`;
    const testId = element.getAttribute('data-testid');
    if (testId) return `[data-testid="${CSS.escape(testId)}"]`;
    const parts: string[] = [];
    let current: Element | null = element;
    while (current && current !== document.body && parts.length < 16) {
      const tag = current.tagName.toLowerCase();
      const parent: Element | null = current.parentElement;
      if (!parent) {
        parts.unshift(tag);
        break;
      }
      const siblings = Array.from(parent.children).filter(
        (sibling) => sibling.tagName === current?.tagName
      );
      parts.unshift(`${tag}:nth-of-type(${siblings.indexOf(current) + 1})`);
      current = parent;
    }
    return parts.join(' > ');
  };
  const elements = Array.from(document.querySelectorAll('[data-rcl-asset]'));
  const components: ComponentIdentity[] = [];
  for (const element of elements) {
    const asset = element.getAttribute('data-rcl-asset') || '';
    const version = element.getAttribute('data-rcl-version') || '';
    const stamp = element.getAttribute('data-rcl-stamp') || '';
    if (!asset || !version) continue;
    const selector = selectorOf(element);
    const sourceSlot = element.getAttribute('data-rcl-source-slot') || undefined;
    components.push({
      selector,
      asset,
      version,
      stamp,
      ...(sourceSlot ? { source_slot: sourceSlot } : {}),
    });
  }
  return {
    schema: 'bas-component-identity/v1',
    element_count: elements.length,
    stamped_count: components.length,
    components,
  };
}
