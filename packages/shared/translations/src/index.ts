declare var require: any
const awardLocale = require('./locales/en/award.json');
const commonLocale = require('./locales/en/common.json');
const errorLocale = require('./locales/en/error.json');
const notifyLocale = require('./locales/en/notify.json');
const validateLocale = require('./locales/en/validate.json');

// Setup internationization
const defaultNS = 'common';
const resources = {
    en: {
        award: awardLocale,
        common: commonLocale,
        validate: validateLocale,
        error: errorLocale,
        notify: notifyLocale,
    },
} as const;

export const i18nConfig = (debug: boolean) => ({
    debug,
    partialBundledLanguages: true,
    defaultNS,
    ns: ['common', 'validation', 'errors', 'notifications'],
    fallbackLng: 'en',
    resources,
    backend: {
        loadPath: './locales/{{lng}}/{{ns}}.json',
    }
})