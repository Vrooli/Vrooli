import { TFuncKey } from "i18next";
import award from "./translations/locales/en/award.json";
import common from "./translations/locales/en/common.json";
import error from "./translations/locales/en/error.json";
import notify from "./translations/locales/en/notify.json";
// import validate from './translations/locales/en/validate.json'

declare module "@local/shared";
export * from ".";

declare module "i18next" {
    interface CustomTypeOptions {
        defaultNS: "common";
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
export type AwardKey = TFuncKey<"award", undefined>
export type CommonKey = TFuncKey<"common", undefined>
export type ErrorKey = TFuncKey<"error", undefined>
export type NotifyKey = TFuncKey<"notify", undefined>
// export type ValidateKey = TFuncKey<'validate', undefined>

export interface SvgProps {
    fill?: string;
    iconTitle?: string;
    id?: string;
    style?: any;
    onClick?: () => any;
    width?: number | string | null;
    height?: number | string | null;
}

export type SvgComponent = (props: SvgProps) => JSX.Element;
