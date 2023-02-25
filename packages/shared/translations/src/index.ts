declare var require: any;
const award = require('./locales/en/award.json');
const common = require('./locales/en/common.json');
const error = require('./locales/en/error.json');
const notify = require('./locales/en/notify.json');
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
    fallbackLng: 'en',
    resources,
    backend: {
        loadPath: './locales/{{lng}}/{{ns}}.json',
    }
})