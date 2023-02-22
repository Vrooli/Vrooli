import { TFuncKey } from 'i18next'
import awardLocale from './locales/en/award.json'
import commonLocale from './locales/en/common.json'
import errorLocale from './locales/en/error.json'
import notifyLocale from './locales/en/notify.json'
import validateLocale from './locales/en/validate.json'

declare module '@shared/route';
export * from '.';

declare module "i18next" {
    interface CustomTypeOptions {
        defaultNS: 'common';
        resources: {
            award: typeof awardLocale;
            common: typeof commonLocale;
            validate: typeof validateLocale;
            error: typeof errorLocale;
            notify: typeof notifyLocale;
        };
    }
}

// Translations
export type AwardKey = TFuncKey<'award', undefined>
export type CommonKey = TFuncKey<'common', undefined>
export type ErrorKey = TFuncKey<'error', undefined>
export type NotifyKey = TFuncKey<'notify', undefined>
export type ValidateKey = TFuncKey<'validate', undefined>