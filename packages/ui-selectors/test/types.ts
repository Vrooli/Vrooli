import { createSelectorRegistry, defineDynamicSelector } from '../src/index.js';
const registry = createSelectorRegistry({ app: { save: 'save' } }, {
  locale: { toggle: defineDynamicSelector({ description: 'locale', testIdPattern: '${code}', params: { code: { type: 'enum', values: ['en', 'ja'] as const } } }) },
}, { dialog: { close: 'close' } });
registry.selectors.locale.toggle({ code: 'en' });
registry.selectors.library.dialog.close;
// @ts-expect-error Unknown locale must fail at the call site.
registry.selectors.locale.toggle({ code: 'xx' });
// @ts-expect-error Required parameters must not disappear during extraction.
registry.selectors.locale.toggle();
// @ts-expect-error Unknown selector paths must fail at the call site.
registry.selectors.app.missing;
