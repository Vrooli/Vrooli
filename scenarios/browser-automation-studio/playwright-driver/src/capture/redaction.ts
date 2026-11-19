/** Browser-side privacy prepass for capture artifacts. Pass redactPageInPage
 * directly to Page.evaluate() before reading pixels, L1 HTML, or L2 text. */
export type RedactionReport = {
  schema: 'bas-capture-redaction/v1';
  status: 'safe' | 'blocked';
  redacted: {
    formValues: number;
    markedRegions: number;
    assignments: number;
    attributes: number;
    urlParameters: number;
    personalData: number;
  };
  unsupported: Array<{ surface: string; reason: string }>;
};

export type RedactPageOptions = {
  /** Child frames are inspected by the Node-side tree walker. */
  inspectChildFrames?: boolean;
};

/** This function must remain closure-free because Playwright serializes it. */
export function redactPageInPage(options?: RedactPageOptions): RedactionReport {
  const doc = document;
  const unsupported: RedactionReport['unsupported'] = [];
  const redacted = {
    formValues: 0,
    markedRegions: 0,
    assignments: 0,
    attributes: 0,
    urlParameters: 0,
    personalData: 0,
  };
  const mask = '[REDACTED]';
  const sensitiveName =
    /(?:password|passwd|passcode|secret|token|otp|one[-_ ]?time|verification[-_ ]?code|cvv|cvc|card[-_ ]?(?:number|security)|api[-_ ]?key|credential|authorization|e[-_ ]?mail|phone|mobile|telephone|address|customer|user[-_ ]?(?:name|id)|account[-_ ]?number|social[-_ ]?security|\bssn\b|date[-_ ]?of[-_ ]?birth|\bdob\b|tax[-_ ]?id)/i;
  // Attribute names that hold a secret or contact value themselves
  // (`data-api-key`, `data-email`), not layout words like `data-mobile-nav`.
  const sensitiveAttributeName =
    /(?:password|passwd|passcode|secret|token|otp|cvv|cvc|api[-_]?key|credential|authorization|e[-_]?mail|\bssn\b|tax[-_]?id)/i;
  const assignment =
    /\b(password|passwd|passcode|secret|token|otp|one[-_ ]?time[-_ ]?code|verification[-_ ]?code|cvv|cvc|api[-_ ]?key|access[-_ ]?token|refresh[-_ ]?token)\b\s*([:=])\s*([^\s,;]+)/gi;
  const email = /\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b/gi;
  const phone = /\b\+?\d[\d ().-]{7,}\d\b/g;
  const redactText = (value: string): string => {
    const keyed = value.replace(assignment, (_match, key: string, separator: string) => {
      redacted.assignments += 1;
      return `${key}${separator}${mask}`;
    });
    const withEmail = keyed.replace(email, () => {
      redacted.personalData += 1;
      return mask;
    });
    return withEmail.replace(phone, () => {
      redacted.personalData += 1;
      return mask;
    });
  };
  // Returns the value unchanged unless it carries credentials, parameters or a
  // fragment, so an ordinary relative link is not rewritten.
  const redactUrl = (value: string): string => {
    if (value.trim().startsWith('#')) return value;
    try {
      const parsed = new URL(value, doc.baseURI);
      let changed = false;
      if (parsed.username || parsed.password) {
        parsed.username = mask;
        parsed.password = mask;
        redacted.urlParameters += 1;
        changed = true;
      }
      for (const key of [...parsed.searchParams.keys()]) {
        parsed.searchParams.set(key, mask);
        redacted.urlParameters += 1;
        changed = true;
      }
      if (parsed.hash) {
        parsed.hash = '#[REDACTED]';
        changed = true;
      }
      return changed ? parsed.href : value;
    } catch {
      return redactText(value);
    }
  };
  // Only navigation targets are rewritten in the live page. Rewriting a
  // resource URL (stylesheet, image, SVG sprite) makes the browser reload or
  // drop it, so the pixels stop showing the app. The frozen page replaces
  // resource URLs with inlined data or redacted diagnostics itself.
  // Text-pattern redaction applies to attributes that carry words for people.
  // Geometry and styling (`viewBox="0 0 24 24"`, path data, class, style) look
  // like phone numbers to the pattern and must never be rewritten.
  const textAttribute = (name: string): boolean =>
    name.startsWith('data-') ||
    [
      'aria-label',
      'aria-description',
      'aria-valuetext',
      'aria-placeholder',
      'title',
      'alt',
      'placeholder',
      'value',
      'label',
      'content',
    ].includes(name);
  const navigationAttribute = (tag: string, name: string): boolean =>
    ((tag === 'a' || tag === 'area') && name === 'href') ||
    name === 'action' ||
    name === 'formaction';
  const elements = Array.from(doc.querySelectorAll('*'));
  // Field names such as "phone" or "address" mark a data-entry control. On any
  // other element they are interface words ("Mobile navigation"), and masking
  // the element would delete visible chrome from the capture.
  const entryControl = (element: Element): boolean =>
    /^(input|textarea|select)$/i.test(element.tagName) ||
    element.getAttribute('contenteditable') === 'true';
  const marked = (element: Element): boolean =>
    element.hasAttribute('data-sensitive') ||
    element.hasAttribute('data-private') ||
    element.getAttribute('data-redact') === 'true' ||
    element.getAttribute('data-capture') === 'redact' ||
    (entryControl(element) &&
      sensitiveName.test(
        [
          element.getAttribute('aria-label') || '',
          element.getAttribute('type') || '',
          element.getAttribute('name') || '',
          element.getAttribute('id') || '',
          element.getAttribute('placeholder') || '',
          element.getAttribute('autocomplete') || '',
          element.getAttribute('title') || '',
        ].join(' ')
      ));
  const hideControl = (element: Element): void => {
    const control = element as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement;
    if ('value' in control && control.value && control.value !== mask) {
      control.value = mask;
      if (element.tagName.toLowerCase() === 'input') {
        element.setAttribute('value', mask);
      } else if (element.tagName.toLowerCase() === 'textarea') {
        element.textContent = mask;
      } else if (element.tagName.toLowerCase() === 'select') {
        const option = doc.createElement('option');
        option.value = mask;
        option.textContent = mask;
        option.selected = true;
        element.replaceChildren(option);
      }
      redacted.formValues += 1;
    }
  };
  const maskedElements = new Set<Element>();
  for (const element of elements) {
    const tag = element.tagName.toLowerCase();
    if (tag === 'canvas')
      unsupported.push({
        surface: 'canvas',
        reason: 'Canvas pixels cannot be inspected by the redaction prepass.',
      });
    // The single-frame fallback cannot see inside a child document. The tree
    // walker passes inspectChildFrames and redacts every frame separately.
    if (tag === 'iframe' && !options?.inspectChildFrames)
      unsupported.push({
        surface: 'iframe',
        reason: 'Embedded frame content is outside the inspected document.',
      });
    if (tag === 'object' || tag === 'embed')
      unsupported.push({
        surface: tag,
        reason: 'Embedded content is outside the inspected document.',
      });
    if (element.shadowRoot)
      unsupported.push({ surface: 'shadow-root', reason: 'Shadow tree content is not inspected.' });
    const control = tag === 'input' || tag === 'textarea' || tag === 'select';
    // Typed values are private; a select shows one of the app's own options.
    if (control && tag !== 'select') hideControl(element);
    for (const attribute of Array.from(element.attributes)) {
      const name = attribute.name.toLowerCase();
      let value = attribute.value;
      if (['href', 'src', 'action', 'formaction', 'xlink:href', 'srcset'].includes(name)) {
        if (navigationAttribute(tag, name)) value = redactUrl(value);
      } else if (sensitiveAttributeName.test(name) && value) value = mask;
      else if (textAttribute(name)) value = redactText(value);
      if (value !== attribute.value) {
        element.setAttribute(attribute.name, value);
        redacted.attributes += 1;
      }
    }
    if (marked(element)) {
      maskedElements.add(element);
      redacted.markedRegions += 1;
      hideControl(element);
      for (const nested of Array.from(element.querySelectorAll('input, textarea, select')))
        hideControl(nested);
      // Replacing the subtree removes any hidden text/attributes from later DOM captures.
      element.textContent = mask;
    }
  }
  const walker = doc.createTreeWalker(doc.documentElement, NodeFilter.SHOW_TEXT);
  let textNode = walker.nextNode();
  while (textNode) {
    const parent = textNode.parentElement;
    if (
      parent &&
      !parent.closest('script, style, noscript, template') &&
      !Array.from(maskedElements).some((region) => region === parent || region.contains(parent))
    ) {
      const before = textNode.nodeValue || '';
      const after = redactText(before);
      if (after !== before) textNode.nodeValue = after;
    }
    textNode = walker.nextNode();
  }
  return {
    schema: 'bas-capture-redaction/v1',
    status: unsupported.length === 0 ? 'safe' : 'blocked',
    redacted,
    unsupported,
  };
}

type CaptureFrame = {
  evaluate<T>(fn: (options?: RedactPageOptions) => T, arg?: RedactPageOptions): Promise<T>;
  childFrames(): CaptureFrame[];
};

type CapturePage = { mainFrame(): CaptureFrame };

/**
 * Redacts the main document and every child frame, then proves the frame tree
 * matches the iframe elements each document declares. Playwright can evaluate
 * inside cross-origin frames even though `contentDocument` is null, so a page
 * that embeds its own harness (or a trusted third-party surface) can produce a
 * complete redaction report instead of being rejected wholesale.
 */
export async function redactPageTree(page: CapturePage): Promise<RedactionReport> {
  const aggregate: RedactionReport = {
    schema: 'bas-capture-redaction/v1',
    status: 'safe',
    redacted: {
      formValues: 0,
      markedRegions: 0,
      assignments: 0,
      attributes: 0,
      urlParameters: 0,
      personalData: 0,
    },
    unsupported: [],
  };
  let index = 0;
  const visit = async (frame: CaptureFrame): Promise<void> => {
    const label = `Embedded frame #${index++}`;
    let report: RedactionReport;
    try {
      report = await frame.evaluate(redactPageInPage, { inspectChildFrames: true });
    } catch {
      aggregate.status = 'blocked';
      aggregate.unsupported.push({
        surface: 'iframe',
        reason: `${label} could not be inspected.`,
      });
      return;
    }
    aggregate.redacted.formValues += report.redacted.formValues;
    aggregate.redacted.markedRegions += report.redacted.markedRegions;
    aggregate.redacted.assignments += report.redacted.assignments;
    aggregate.redacted.attributes += report.redacted.attributes;
    aggregate.redacted.urlParameters += report.redacted.urlParameters;
    aggregate.redacted.personalData += report.redacted.personalData;
    aggregate.unsupported.push(...report.unsupported);
    if (report.status === 'blocked') aggregate.status = 'blocked';

    const declared = await frame
      .evaluate(() => document.querySelectorAll('iframe, frame').length)
      .catch(() => -1);
    const children = frame.childFrames();
    if (declared !== children.length) {
      aggregate.status = 'blocked';
      aggregate.unsupported.push({
        surface: 'iframe',
        reason: `${label} declares ${declared < 0 ? 'unknown' : declared} embedded frame(s) but only ${children.length} are inspectable.`,
      });
    }
    for (const child of children) await visit(child);
  };
  await visit(page.mainFrame());
  return aggregate;
}
