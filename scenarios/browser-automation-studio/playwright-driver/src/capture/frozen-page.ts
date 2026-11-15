/** L1 frozen-page capture. The collector is closure-free and can be injected with
 * Playwright page.evaluate(); the serializer also accepts supplied snapshots for
 * deterministic callers and tests. */

export type FrozenAsset = { url: string; dataUrl?: string; reason?: string };
export type FrozenFrameTile = { label: string; dataUrl: string; reason?: string };
export type FrozenPageSnapshot = {
  url: string;
  title: string;
  html: string;
  css: string;
  assets: FrozenAsset[];
  frameTiles?: FrozenFrameTile[];
  unsupported: Array<{ surface: string; reason: string }>;
};
export type FrozenPageResult = {
  schema: 'bas-frozen-page/v1';
  html: string;
  status: 'present' | 'partial';
  bytes: number;
  byteBudget: number;
  reasons: string[];
  unsupported: Array<{ surface: string; reason: string }>;
};
export type FrozenPageOptions = { byteBudget?: number };

/** Collects rendered DOM and accessible stylesheet rules without executing page scripts. */
export async function collectFrozenPageInPage(): Promise<FrozenPageSnapshot> {
  const doc = document;
  const unsupported: Array<{ surface: string; reason: string }> = [];
  // This helper is repeated inside the injected collector because Playwright
  // serializes the function body and drops module closures.
  const safeDiagnosticUrlInPage = (raw: string): string => {
    try {
      const parsed = new URL(raw, doc.baseURI);
      parsed.username = '';
      parsed.password = '';
      for (const key of [...parsed.searchParams.keys()]) parsed.searchParams.set(key, '[REDACTED]');
      if (parsed.hash) parsed.hash = '#[REDACTED]';
      return parsed.href;
    } catch {
      return '[URL REDACTED]';
    }
  };
  const clone = doc.documentElement.cloneNode(true) as HTMLElement;
  for (const node of Array.from(
    clone.querySelectorAll('script, noscript, object, embed, applet, template')
  ))
    node.remove();
  // Styles are re-emitted below as one pruned sheet; other links (preload,
  // icons, manifests) cannot load offline and may carry private parameters.
  for (const node of Array.from(clone.querySelectorAll('style, link'))) node.remove();
  // cloneNode copies attributes, not live form state: carry the chosen option,
  // checked boxes and (already redacted) typed values into the frozen markup.
  const liveControls = Array.from(doc.querySelectorAll('select, input, textarea'));
  const clonedControls = Array.from(clone.querySelectorAll('select, input, textarea'));
  liveControls.forEach((live, index) => {
    const copy = clonedControls[index];
    if (!copy) return;
    if (live.tagName === 'SELECT')
      Array.from(copy.querySelectorAll('option')).forEach((option, optionIndex) => {
        if (optionIndex === (live as HTMLSelectElement).selectedIndex)
          option.setAttribute('selected', '');
        else option.removeAttribute('selected');
      });
    else if (live.tagName === 'INPUT') {
      const input = live as HTMLInputElement;
      if (input.type === 'checkbox' || input.type === 'radio') {
        if (input.checked) copy.setAttribute('checked', '');
        else copy.removeAttribute('checked');
      } else if (input.type !== 'password' && input.type !== 'file')
        copy.setAttribute('value', input.value);
    } else if (live.tagName === 'TEXTAREA') copy.textContent = (live as HTMLTextAreaElement).value;
  });
  const comments = doc.createTreeWalker(clone, NodeFilter.SHOW_COMMENT);
  let comment = comments.nextNode();
  while (comment) {
    const next = comments.nextNode();
    comment.parentNode?.removeChild(comment);
    comment = next;
  }
  for (const element of Array.from(clone.querySelectorAll('*'))) {
    for (const attribute of Array.from(element.attributes)) {
      const name = attribute.name.toLowerCase();
      if (name.startsWith('on') || name === 'srcdoc' || name === 'nonce')
        element.removeAttribute(attribute.name);
      if (
        ['href', 'src', 'action', 'formaction', 'xlink:href'].includes(name) &&
        /^\s*javascript:/i.test(attribute.value)
      )
        element.removeAttribute(attribute.name);
      else if (
        ['href', 'src', 'xlink:href', 'poster'].includes(name) &&
        /[?#@]/.test(attribute.value) &&
        !/^\s*(#|data:)/i.test(attribute.value)
      )
        // The live page keeps resource URLs intact for its pixels; the frozen
        // copy drops their parameters, fragments and credentials.
        element.setAttribute(attribute.name, safeDiagnosticUrlInPage(attribute.value));
      if (name === 'srcset') element.removeAttribute(attribute.name);
    }
  }

  const usedFonts = new Set<string>();
  const usedAnimations = new Set<string>();
  const styledElements = Array.from(doc.querySelectorAll('body *'));
  if (styledElements.length > 5000)
    unsupported.push({
      surface: 'css-usage',
      reason: 'Font and animation usage was sampled for 5,000 of more than 5,000 elements.',
    });
  for (const element of styledElements.slice(0, 5000)) {
    const computed = getComputedStyle(element);
    for (const family of computed.fontFamily.split(','))
      usedFonts.add(
        family
          .trim()
          .replace(/^['"]|['"]$/g, '')
          .toLowerCase()
      );
    for (const name of computed.animationName.split(','))
      if (name.trim() && name.trim() !== 'none') usedAnimations.add(name.trim());
  }
  const isUsedSelector = (selector: string): boolean => {
    try {
      // Match the element a pseudo-element hangs off. A part that is only a
      // pseudo-element (`*, ::before, ::after`) applies to every element.
      const query = selector
        .split(/,(?![^(]*\))/)
        .map(
          (part) =>
            part
              .replace(/::[\w-]+(\([^)]*\))?/g, '')
              .replace(/:(before|after|first-line|first-letter)\b/gi, '')
              .trim() || '*'
        )
        .join(', ');
      return Array.from(doc.querySelectorAll(query)).some((element) => element.isConnected);
    } catch {
      unsupported.push({
        surface: `css:${selector}`,
        reason: 'selector could not be evaluated safely',
      });
      return false;
    }
  };
  const collectRules = (source: CSSRuleList): string[] => {
    const selected: string[] = [];
    for (const rule of Array.from(source)) {
      if (rule instanceof CSSStyleRule) {
        if (isUsedSelector(rule.selectorText)) selected.push(rule.cssText);
        continue;
      }
      if (rule instanceof CSSFontFaceRule) {
        const family = rule.style
          .getPropertyValue('font-family')
          .trim()
          .replace(/^['"]|['"]$/g, '')
          .toLowerCase();
        if (family && usedFonts.has(family)) selected.push(rule.cssText);
        continue;
      }
      if (rule instanceof CSSKeyframesRule) {
        if (usedAnimations.has(rule.name)) selected.push(rule.cssText);
        continue;
      }
      if (rule instanceof CSSMediaRule) {
        if (!matchMedia(rule.conditionText).matches) continue;
        const nested = collectRules(rule.cssRules);
        if (nested.length) selected.push(`@media ${rule.conditionText} {\n${nested.join('\n')}\n}`);
        continue;
      }
      if (rule instanceof CSSSupportsRule) {
        if (!CSS.supports(rule.conditionText)) continue;
        const nested = collectRules(rule.cssRules);
        if (nested.length)
          selected.push(`@supports ${rule.conditionText} {\n${nested.join('\n')}\n}`);
        continue;
      }
      if (rule instanceof CSSImportRule) {
        try {
          if (rule.styleSheet) selected.push(...collectRules(rule.styleSheet.cssRules));
        } catch {
          unsupported.push({
            surface: `css-import:${safeDiagnosticUrlInPage(rule.href)}`,
            reason: 'imported stylesheet is inaccessible',
          });
        }
        continue;
      }
      const nested = (rule as CSSGroupingRule).cssRules;
      if (nested) {
        const selectedChildren = collectRules(nested);
        if (selectedChildren.length) {
          const prelude = rule.cssText.slice(0, rule.cssText.indexOf('{')).trim();
          selected.push(`${prelude} {\n${selectedChildren.join('\n')}\n}`);
        }
        continue;
      }
      if (rule instanceof CSSNamespaceRule) selected.push(rule.cssText);
      else
        unsupported.push({
          surface: `css-rule:${rule.constructor.name}`,
          reason: 'CSS rule type is not handled by the used-rule collector',
        });
    }
    return selected;
  };
  const rules: string[] = [];
  for (const sheet of Array.from(doc.styleSheets)) {
    try {
      rules.push(...collectRules(sheet.cssRules));
    } catch {
      unsupported.push({
        surface: `stylesheet:${sheet.href ? safeDiagnosticUrlInPage(sheet.href) : 'inline'}`,
        reason: 'cross-origin or inaccessible CSS rules',
      });
    }
  }

  const assets: FrozenAsset[] = [];
  const inline = async (url: string): Promise<string | undefined> => {
    try {
      const safeFetchUrl = new URL(url, doc.baseURI);
      safeFetchUrl.username = '';
      safeFetchUrl.password = '';
      for (const key of [...safeFetchUrl.searchParams.keys()])
        safeFetchUrl.searchParams.set(key, '[REDACTED]');
      if (safeFetchUrl.hash) safeFetchUrl.hash = '';
      const response = await fetch(safeFetchUrl.href, { credentials: 'same-origin' });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const blob = await response.blob();
      const dataUrl = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () =>
          typeof reader.result === 'string'
            ? resolve(reader.result)
            : reject(new Error('not a data URL'));
        reader.onerror = () => reject(new Error('asset read failed'));
        reader.readAsDataURL(blob);
      });
      assets.push({ url: safeDiagnosticUrlInPage(url), dataUrl });
      return dataUrl;
    } catch (error) {
      const reason = error instanceof Error ? error.message : 'asset could not be fetched';
      assets.push({ url, reason });
      unsupported.push({
        surface: `asset:${safeDiagnosticUrlInPage(url)}`,
        reason: `not inlined: ${reason}`,
      });
      return undefined;
    }
  };
  const liveImages = Array.from(doc.images);
  const clonedImages = Array.from(clone.querySelectorAll('img'));
  for (let index = 0; index < liveImages.length; index++) {
    const image = liveImages[index];
    const copy = clonedImages[index];
    if (!image || !copy) continue;
    const source = image.currentSrc || image.src;
    if (!source) continue;
    // Inline at twice the drawn size, not the file's size: a 7 MB photo drawn
    // as a 64 px thumbnail otherwise blows the page budget.
    const box = image.getBoundingClientRect();
    const scale = Math.min(
      1,
      2 *
        Math.max(
          Math.max(box.width, 32) / Math.max(image.naturalWidth, 1),
          Math.max(box.height, 32) / Math.max(image.naturalHeight, 1)
        )
    );
    if (image.complete && image.naturalWidth > 0 && scale < 0.9) {
      try {
        const canvas = doc.createElement('canvas');
        canvas.width = Math.max(1, Math.round(image.naturalWidth * scale));
        canvas.height = Math.max(1, Math.round(image.naturalHeight * scale));
        canvas.getContext('2d')?.drawImage(image, 0, 0, canvas.width, canvas.height);
        const scaled = canvas.toDataURL('image/webp', 0.86);
        if (scaled.startsWith('data:image/webp')) {
          assets.push({ url: safeDiagnosticUrlInPage(source), dataUrl: scaled });
          copy.setAttribute('src', scaled);
          copy.removeAttribute('srcset');
          copy.removeAttribute('sizes');
          continue;
        }
      } catch {
        /* cross-origin pixels are tainted; fall back to the original file */
      }
    }
    const data = await inline(source);
    copy.setAttribute('src', data ?? safeDiagnosticUrlInPage(source));
    copy.removeAttribute('srcset');
    copy.removeAttribute('sizes');
  }
  // Keep frame content out of markup and leave an indexed slot for a sanitized screenshot tile.
  for (const [index, frame] of Array.from(clone.querySelectorAll('iframe, frame')).entries()) {
    frame.replaceWith(doc.createComment(`bas-frame:${index}`));
  }
  const liveCanvases = Array.from(doc.querySelectorAll('canvas'));
  const clonedCanvases = Array.from(clone.querySelectorAll('canvas'));
  for (let index = 0; index < liveCanvases.length; index++) {
    const canvas = liveCanvases[index];
    const copy = clonedCanvases[index];
    if (!canvas || !copy) continue;
    try {
      const image = doc.createElement('img');
      image.alt = 'Frozen canvas rendering';
      image.width = canvas.width;
      image.height = canvas.height;
      image.src = canvas.toDataURL('image/png');
      copy.replaceWith(image);
    } catch {
      unsupported.push({
        surface: 'canvas',
        reason: 'canvas pixels are tainted or unavailable; rendering omitted',
      });
    }
  }
  let css = rules.join('\n');
  for (const match of Array.from(css.matchAll(/url\((['"]?)(.*?)\1\)/g))) {
    const raw = match[2];
    if (!raw || raw.startsWith('data:') || raw.startsWith('#')) continue;
    const data = await inline(raw);
    css = css.replace(match[0], `url("${data ?? safeDiagnosticUrlInPage(raw)}")`);
  }
  const style = doc.createElement('style');
  style.textContent = css;
  clone.querySelector('head')?.appendChild(style);
  const doctype = doc.doctype ? `<!DOCTYPE ${doc.doctype.name}>` : '<!DOCTYPE html>';
  return {
    url: doc.URL,
    title: doc.title,
    html: `${doctype}${clone.outerHTML}`,
    css,
    assets,
    unsupported,
  };
}

/** Applies a UTF-8 byte budget; oversized captures remain explicitly partial. */
export function serializeFrozenPage(
  snapshot: FrozenPageSnapshot,
  options: FrozenPageOptions = {}
): FrozenPageResult {
  const budget = Math.max(256, Math.floor(options.byteBudget ?? 5_000_000));
  const reasons: string[] = [];
  let html = snapshot.html;
  const unsupported = [...snapshot.unsupported];
  const bytes = (value: string): number => new TextEncoder().encode(value).byteLength;
  let size = bytes(html);
  if (size > budget) {
    html = '';
    size = 0;
    reasons.push(
      'Document not emitted because it exceeded the byte budget; other capture lenses can still be present.'
    );
    unsupported.push({ surface: 'document', reason: 'document not emitted: byte budget exceeded' });
  }
  if (html) {
    const tiles = snapshot.frameTiles ?? [];
    const placeholderCount = Array.from(html.matchAll(/<!--bas-frame:(\d+)-->/g)).length;
    for (let index = 0; index < placeholderCount; index++) {
      const marker = `<!--bas-frame:${index}-->`;
      const tile = tiles[index];
      const label = tile?.label || `Embedded frame ${index + 1}`;
      const unavailable =
        tile?.reason || (tile?.dataUrl ? undefined : 'frame screenshot unavailable');
      const replacement = unavailable
        ? `<figure class="bas-frame-tile"><figcaption>${escapeHtml(label)} · screenshot unavailable: ${escapeHtml(unavailable)}</figcaption></figure>`
        : `<figure class="bas-frame-tile"><figcaption>${escapeHtml(label)} · captured pixels</figcaption><img alt="${escapeHtml(label)} screenshot" src="${tile?.dataUrl ?? ''}"></figure>`;
      const candidate = html.replace(marker, replacement);
      if (!unavailable && bytes(candidate) > budget) {
        unsupported.push({
          surface: label,
          reason: 'frame screenshot omitted: aggregate byte budget exceeded',
        });
        const omitted = `<figure class="bas-frame-tile"><figcaption>${escapeHtml(label)} · screenshot omitted: aggregate byte budget exceeded</figcaption></figure>`;
        const omittedHtml = html.replace(marker, omitted);
        html = bytes(omittedHtml) <= budget ? omittedHtml : html.replace(marker, '');
      } else if (bytes(candidate) <= budget) {
        html = candidate;
        if (unavailable) unsupported.push({ surface: label, reason: unavailable });
      } else {
        // Keep the indexed comment if even an explanatory placeholder would exceed the limit.
        unsupported.push({
          surface: label,
          reason: unavailable || 'frame placeholder omitted: aggregate byte budget exceeded',
        });
      }
      size = bytes(html);
    }
    for (const tile of tiles.slice(placeholderCount)) {
      unsupported.push({
        surface: tile.label,
        reason: tile.reason || 'frame screenshot omitted: no matching document placeholder',
      });
    }
  }
  for (const asset of snapshot.assets)
    if (asset.reason)
      unsupported.push({ surface: `asset:${safeDiagnosticUrl(asset.url)}`, reason: asset.reason });
  if (unsupported.length)
    reasons.push('One or more page surfaces or assets could not be frozen completely.');
  return {
    schema: 'bas-frozen-page/v1',
    html,
    status: reasons.length ? 'partial' : 'present',
    bytes: size,
    byteBudget: budget,
    reasons,
    unsupported,
  };
}

function escapeHtml(value: string): string {
  return value.replace(
    /[&<>"']/g,
    (character) =>
      ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[character] ??
      character
  );
}

function safeDiagnosticUrl(raw: string): string {
  try {
    const parsed = new URL(raw, 'https://capture.invalid/');
    parsed.username = '';
    parsed.password = '';
    for (const key of [...parsed.searchParams.keys()]) parsed.searchParams.set(key, '[REDACTED]');
    if (parsed.hash) parsed.hash = '#[REDACTED]';
    return parsed.href.replace('https://capture.invalid/', '/');
  } catch {
    return '[URL REDACTED]';
  }
}
