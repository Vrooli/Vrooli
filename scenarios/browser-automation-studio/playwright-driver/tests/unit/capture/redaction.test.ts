import { redactPageInPage } from '../../../src/capture/redaction';

type FakeText = { nodeType: 3; nodeValue: string; parentElement: FakeElement };
type FakeElement = {
  tagName: string;
  attrs: Record<string, string>;
  attributes: Array<{ name: string; value: string }>;
  value: string;
  children: FakeElement[];
  texts: FakeText[];
  shadowRoot: null;
  textContent: string | null;
  hasAttribute(name: string): boolean;
  getAttribute(name: string): string | null;
  setAttribute(name: string, value: string): void;
  querySelectorAll(selector: string): FakeElement[];
  contains(other: FakeElement): boolean;
  closest(selector: string): FakeElement | null;
};

function el(tagName: string, attrs: Record<string, string> = {}, value = ''): FakeElement {
  const result: FakeElement = {
    tagName: tagName.toUpperCase(),
    attrs,
    get attributes() {
      return Object.entries(this.attrs).map(([name, value]) => ({ name, value }));
    },
    value,
    children: [],
    texts: [],
    shadowRoot: null,
    textContent: null,
    hasAttribute(name) {
      return Object.hasOwn(this.attrs, name);
    },
    getAttribute(name) {
      return this.attrs[name] ?? null;
    },
    setAttribute(name, value) {
      this.attrs[name] = value;
    },
    querySelectorAll(selector) {
      const tags = selector.split(',').map((part) => part.trim().toUpperCase());
      return this.children.flatMap((child) => [
        ...(tags.includes('*') || tags.includes(child.tagName) ? [child] : []),
        ...child.querySelectorAll(selector),
      ]);
    },
    contains(other) {
      return this.children.some((child) => child === other || child.contains(other));
    },
    closest() {
      return null;
    },
  };
  return result;
}

function text(parent: FakeElement, nodeValue: string): FakeText {
  const node = { nodeType: 3 as const, nodeValue, parentElement: parent };
  parent.texts.push(node);
  return node;
}

function install(body: FakeElement): () => void {
  const oldDocument = globalThis.document;
  const oldFilter = globalThis.NodeFilter;
  const all: FakeElement[] = [];
  const visit = (item: FakeElement): void => {
    all.push(item);
    item.children.forEach(visit);
  };
  visit(body);
  globalThis.NodeFilter = { SHOW_TEXT: 4 } as typeof NodeFilter;
  globalThis.document = {
    body,
    documentElement: body,
    baseURI: 'https://example.test/',
    querySelectorAll: () => all,
    createTreeWalker: () => {
      const nodes = all.flatMap((item) => item.texts);
      let index = 0;
      return { nextNode: () => nodes[index++] ?? null };
    },
  } as unknown as Document;
  return () => {
    globalThis.document = oldDocument;
    globalThis.NodeFilter = oldFilter;
  };
}

describe('capture redaction prepass', () => {
  it('redacts password/token controls, marked regions, and inline assignments without reporting values', () => {
    const body = el('body');
    const password = el('input', { type: 'password', name: 'password' }, 'pw-raw-777');
    const token = el('textarea', { 'aria-label': 'Access token' }, 'tok-raw-888');
    const region = el('section', { 'data-sensitive': '' });
    const privateLink = el('a', {
      href: 'https://alice:secret@example.test/path?user=alice#private-state',
    });
    const head = el('head');
    const title = el('title');
    const titleText = text(title, 'alice@example.test private page');
    head.children.push(title);
    text(region, 'private region raw');
    const assignment = el('p');
    const assignmentText = text(assignment, 'token=secret-raw-999 status=ok');
    body.children.push(head, password, token, region, assignment, privateLink);
    const restore = install(body);
    try {
      const report = redactPageInPage();
      expect(report.status).toBe('safe');
      expect(report.redacted).toEqual({
        formValues: 2,
        markedRegions: 3,
        assignments: 1,
        attributes: 1,
        urlParameters: 2,
        personalData: 1,
      });
      expect(password.value).toBe('[REDACTED]');
      expect(token.value).toBe('[REDACTED]');
      expect(region.textContent).toBe('[REDACTED]');
      expect(assignmentText.nodeValue).toBe('token=[REDACTED] status=ok');
      expect(privateLink.attrs.href).not.toMatch(/alice|secret|private-state/);
      expect(titleText.nodeValue).toBe('[REDACTED] private page');
      expect(JSON.stringify(report)).not.toMatch(/raw|secret-raw|private-state/);
    } finally {
      restore();
    }
  });

  it('blocks artifact reads when canvas or an unknown embedded surface cannot be inspected', () => {
    const body = el('body');
    body.children.push(el('canvas'), el('iframe'));
    const restore = install(body);
    try {
      const report = redactPageInPage();
      expect(report.status).toBe('blocked');
      expect(report.unsupported.map((item) => item.surface)).toEqual(['canvas', 'iframe']);
    } finally {
      restore();
    }
  });
});

describe('redaction keeps interface chrome intact', () => {
  it('does not mask layout words, geometry or resource URLs, but still masks a phone field', () => {
    const body = el('body');
    const nav = el('nav', { 'aria-label': 'Mobile navigation', 'data-mobile-nav': 'tabs' });
    text(nav, 'Home');
    const icon = el('svg', { viewBox: '0 0 24 24', class: 'h-4 w-4' });
    const path = el('path', { d: 'M3 3h7v7H3z M14 3 2 2 0 0 1-2 2' });
    icon.children.push(path);
    const sprite = el('use', { href: '#icon-home' });
    const stylesheet = el('link', { rel: 'stylesheet', href: '/assets/index.css?v=12' });
    const theme = el('select', { 'aria-label': 'Theme' }, 'system');
    const phone = el('input', { type: 'tel', name: 'phone' }, '+1 555 010 9999');
    body.children.push(nav, icon, sprite, stylesheet, theme, phone);
    const restore = install(body);
    try {
      const report = redactPageInPage();
      expect(nav.textContent).toBeNull();
      expect(nav.attrs['data-mobile-nav']).toBe('tabs');
      expect(icon.attrs.viewBox).toBe('0 0 24 24');
      expect(path.attrs.d).toBe('M3 3h7v7H3z M14 3 2 2 0 0 1-2 2');
      expect(sprite.attrs.href).toBe('#icon-home');
      expect(stylesheet.attrs.href).toBe('/assets/index.css?v=12');
      expect(theme.value).toBe('system');
      expect(phone.value).toBe('[REDACTED]');
      expect(report.redacted.markedRegions).toBe(1);
    } finally {
      restore();
    }
  });
});
