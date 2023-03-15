import { TFuncKey } from 'i18next'
import award from './locales/en/award.json'
import common from './locales/en/common.json'
import error from './locales/en/error.json'
import notify from './locales/en/notify.json'
// import validate from './locales/en/validate.json'

declare module '@shared/route';
export * from '.';

declare module "i18next" {
    interface CustomTypeOptions {
        defaultNS: 'common';
        resources: {
            award: typeof award;
            common: typeof common;
            // validate: typeof validate;
            error: typeof error;
            notify: typeof notify;
        };
    }
}

// Translations
export type AwardKey = TFuncKey<'award', undefined>
export type CommonKey = TFuncKey<'common', undefined>
export type ErrorKey = TFuncKey<'error', undefined>
export type NotifyKey = TFuncKey<'notify', undefined>
// export type ValidateKey = TFuncKey<'validate', undefined>