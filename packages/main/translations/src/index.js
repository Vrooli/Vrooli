import award from './locales/en/award.json';
import common from './locales/en/common.json';
import error from './locales/en/error.json';
import notify from './locales/en/notify.json';
const defaultNS = 'common';
const resources = {
    en: {
        award,
        common,
        error,
        notify,
    },
};
export const i18nConfig = (debug) => ({
    debug,
    partialBundledLanguages: true,
    defaultNS,
    ns: ['common', 'error', 'notify', 'award'],
    nsSeparator: ':',
    fallbackLng: 'en',
    resources,
    backend: {
        loadPath: './locales/{{lng}}/{{ns}}.json',
    },
});
//# sourceMappingURL=index.js.map