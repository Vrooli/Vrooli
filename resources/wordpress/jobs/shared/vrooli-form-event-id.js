(() => {
  'use strict';

  const makeID = () => {
    if (globalThis.crypto && typeof globalThis.crypto.randomUUID === 'function') {
      return globalThis.crypto.randomUUID();
    }
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, token => {
      const random = Math.random() * 16 | 0;
      return (token === 'x' ? random : (random & 0x3 | 0x8)).toString(16);
    });
  };

  const inputsFor = root => {
    const marked = root.matches && root.matches('.vrooli-event-id')
      ? [root]
      : Array.from(root.querySelectorAll ? root.querySelectorAll('.vrooli-event-id') : []);
    return marked.flatMap(node => node.matches('input,textarea') ? [node] : Array.from(node.querySelectorAll('input,textarea')));
  };

  const assignMissing = root => {
    inputsFor(root).forEach(input => {
      if (!input.value.trim()) input.value = makeID();
    });
  };

  const renew = form => {
    inputsFor(form).forEach(input => { input.value = makeID(); });
  };

  const bindSuccessfulRotations = root => {
    root.querySelectorAll('form.wpcf7-form, form.wpforms-form').forEach(form => {
      if (form.dataset.vrooliEventBound) return;
      form.dataset.vrooliEventBound = 'true';
      form.addEventListener('wpcf7mailsent', () => renew(form));
      if (globalThis.jQuery) {
        globalThis.jQuery(form).on('wpformsAjaxSubmitSuccess.vrooliEventId', () => renew(form));
      }
    });
  };

  const run = root => {
    assignMissing(root);
    bindSuccessfulRotations(root);
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => run(document), { once: true });
  } else {
    run(document);
  }
  new MutationObserver(records => records.forEach(record => record.addedNodes.forEach(node => {
    if (node.nodeType === Node.ELEMENT_NODE) run(node);
  }))).observe(document.documentElement, { childList: true, subtree: true });
})();
