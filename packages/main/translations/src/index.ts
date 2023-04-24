import award from './locales/en/award.json';
import common from './locales/en/common.json';
import error from './locales/en/error.json';
import notify from './locales/en/notify.json';
// const validate = require('./locales/en/validate.json');

// Setup internationization
const defaultNS = 'common';
const resources = {
    en: {
        award,
        common,
        // validate,
        error,
        notify,
    },
} as const;

export const i18nConfig = (debug: boolean) => ({
    debug,
    partialBundledLanguages: true,
    defaultNS,
    ns: ['common', 'error', 'notify', 'award'], // 'validate'
    nsSeparator: ':',
    fallbackLng: 'en',
    resources,
    backend: {
        loadPath: './locales/{{lng}}/{{ns}}.json',
    },
})