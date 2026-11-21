import { collectOutlineInPage } from '../../../src/capture/outline';

type MockElement = {
  tagName: string;
  className: string;
  children: MockElement[];
  parentElement: MockElement | null;
  textContent: string;
  attrs: Record<string, string>;
  disabled: boolean;
  shadowRoot: null;
  getAttribute(name: string): string | null;
  setAttribute(name: string, value: string): void;
  hasAttribute(name: string): boolean;
  getBoundingClientRect(): { x: number; y: number; width: number; height: number };
  querySelectorAll(selector: string): MockElement[];
};

function element(tagName: string, text = '', attrs: Record<string, string> = {}): MockElement {
  const result: MockElement = {
    tagName: tagName.toUpperCase(),
    className: attrs.class || '',
    children: [],
    parentElement: null,
    textContent: text,
    attrs,
    disabled: false,
    shadowRoot: null,
    getAttribute(name) {
      return this.attrs[name] ?? null;
    },
    setAttribute(name, value) {
      this.attrs[name] = value;
    },
    hasAttribute(name) {
      return Object.prototype.hasOwnProperty.call(this.attrs, name);
    },
    getBoundingClientRect() {
      return { x: 1, y: 2, width: 120, height: 30 };
    },
    querySelectorAll(selector) {
      const matches = (candidate: MockElement): boolean =>
        selector === '*' ||
        selector.split(',').some((part) => {
          const normalized = part.trim();
          if (normalized.startsWith('[role="'))
            return candidate.attrs.role === normalized.slice(7, -2);
          if (normalized === '[tabindex]') return candidate.hasAttribute('tabindex');
          return candidate.tagName.toLowerCase() === normalized;
        });
      const found: MockElement[] = [];
      const visit = (parent: MockElement): void => {
        for (const child of parent.children) {
          if (matches(child)) found.push(child);
          visit(child);
        }
      };
      visit(this);
      return found;
    },
  };
  return result;
}

function append(parent: MockElement, child: MockElement): void {
  child.parentElement = parent;
  parent.children.push(child);
}

function installDocument(body: MockElement): () => void {
  const previousDocument = globalThis.document;
  const previousStyle = globalThis.getComputedStyle;
  const previousCSS = globalThis.CSS;
  const all = (): MockElement[] => body.querySelectorAll('*');
  globalThis.document = {
    body,
    querySelectorAll: (selector: string) =>
      selector === 'body *'
        ? all()
        : selector === '*'
          ? all()
          : all().filter((item) =>
              selector.split(',').some((tag) => item.tagName.toLowerCase() === tag.trim())
            ),
    querySelector: (selector: string) =>
      all().find((item) =>
        selector.split(',').some((tag) => item.tagName.toLowerCase() === tag.trim())
      ) ?? null,
  } as unknown as Document;
  globalThis.getComputedStyle = (() => ({
    display: 'block',
    visibility: 'visible',
    opacity: '1',
    animationName: 'none',
    transitionDuration: '0s',
  })) as typeof getComputedStyle;
  globalThis.CSS = { ...globalThis.CSS, escape: (value: string) => value };
  return () => {
    globalThis.document = previousDocument;
    globalThis.getComputedStyle = previousStyle;
    globalThis.CSS = previousCSS;
  };
}

describe('collectOutlineInPage', () => {
  it('retains every action when the text budget is exceeded', () => {
    const body = element('body');
    const main = element('main');
    append(body, main);
    for (let index = 0; index < 8; index++)
      append(main, element('button', `Action ${index} with a long descriptive label`));
    const restore = installDocument(body);
    try {
      const result = collectOutlineInPage({ tokenBudget: 5 });
      const actions = result.nodes.filter((node) => node.kind === 'action');
      expect(result.actionCount).toBe(8);
      expect(actions).toHaveLength(8);
      expect(actions.map((node) => node.id)).toEqual([...new Set(actions.map((node) => node.id))]);
      expect(result.overBudget).toBe(true);
      expect(result.estimatedTokens).toBe(
        Math.ceil(new TextEncoder().encode(JSON.stringify(result.nodes)).byteLength / 3)
      );
    } finally {
      restore();
    }
  });

  it('stamps node ids onto elements when asked, so a frozen page is editable by id', () => {
    const body = element('body');
    const main = element('main');
    append(body, main);
    append(main, element('button', 'Save'));
    const restore = installDocument(body);
    try {
      const result = collectOutlineInPage({ stampAttribute: 'data-ux-node' });
      const save = result.nodes.find((node) => node.kind === 'action');
      expect(save).toBeDefined();
      expect(main.children[0]?.getAttribute('data-ux-node')).toBe(save?.id);
      expect(collectOutlineInPage({}).nodes.length).toBe(result.nodes.length);
    } finally {
      restore();
    }
  });

  it('includes custom ARIA controls that do not use native action elements', () => {
    const body = element('body');
    const main = element('main');
    append(body, main);
    append(main, element('div', 'Choose option', { role: 'menuitemcheckbox' }));
    append(main, element('div', 'Expand details', { 'aria-expanded': 'false' }));
    const restore = installDocument(body);
    try {
      const result = collectOutlineInPage();
      const actions = result.nodes.filter((node) => node.kind === 'action');
      expect(result.actionCount).toBe(2);
      expect(actions.map((node) => node.text)).toEqual(['Choose option', 'Expand details']);
    } finally {
      restore();
    }
  });

  it('collapses repeated siblings into a template with examples and keeps descendant actions', () => {
    const body = element('body');
    const main = element('main');
    append(body, main);
    for (let index = 0; index < 4; index++) {
      const card = element('article', `Card ${index}`, { class: 'card' });
      append(card, element('button', `Open ${index}`));
      append(main, card);
    }
    const restore = installDocument(body);
    try {
      const result = collectOutlineInPage();
      const templates = result.nodes.filter((node) => node.kind === 'repeated-template');
      expect(templates).toHaveLength(1);
      expect(templates[0]?.examples).toHaveLength(3);
      expect(result.actionCount).toBe(4);
      expect(templates[0]?.actionIds).toHaveLength(4);
    } finally {
      restore();
    }
  });
});
