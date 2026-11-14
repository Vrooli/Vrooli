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

/** This function must remain closure-free because Playwright serializes it. */
export function redactPageInPage(): RedactionReport {
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
  const redactUrl = (value: string): string => {
    try {
      const parsed = new URL(value, doc.baseURI);
      if (parsed.username || parsed.password) {
        parsed.username = mask;
        parsed.password = mask;
        redacted.urlParameters += 1;
      }
      for (const key of [...parsed.searchParams.keys()]) {
        parsed.searchParams.set(key, mask);
        redacted.urlParameters += 1;
      }
      if (parsed.hash) parsed.hash = '#[REDACTED]';
      return parsed.href;
    } catch {
      return redactText(value);
    }
  };
  const elements = Array.from(doc.querySelectorAll('*'));
  const marked = (element: Element): boolean =>
    element.hasAttribute('data-sensitive') ||
    element.hasAttribute('data-private') ||
    element.getAttribute('data-redact') === 'true' ||
    element.getAttribute('data-capture') === 'redact' ||
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
    );
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
    if (tag === 'iframe')
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
    if (control) hideControl(element);
    for (const attribute of Array.from(element.attributes)) {
      const name = attribute.name.toLowerCase();
      let value = attribute.value;
      if (['href', 'src', 'action', 'formaction', 'xlink:href'].includes(name)) {
        value = redactUrl(value);
      } else {
        if (sensitiveName.test(name) && value) value = mask;
        else value = redactText(value);
      }
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
