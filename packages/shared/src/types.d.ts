import { TFuncKey } from "i18next";
import award from "./translations/locales/en/award.json" assert { type: "json" };
import common from "./translations/locales/en/common.json" assert { type: "json" };
import error from "./translations/locales/en/error.json" assert { type: "json" };
import langs from "./translations/locales/en/langs.json" assert { type: "json" };
import notify from "./translations/locales/en/notify.json" assert { type: "json" };
// import validate from "./translations/locales/en/validate.json" assert { type: "json" };
import "yup";

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
            langs: typeof langs;
            notify: typeof notify;
        };
    }
}

declare module "yup" {
    interface StringSchema {
        /** Converts empty/whitespace strings to undefined */
        removeEmptyString(): this;
    }
    interface BooleanSchema {
        /** Converts data to boolean */
        toBool(): this;
    }
}

// Translations
export type AwardKey = TFuncKey<"award", undefined>
export type CommonKey = TFuncKey<"common", undefined>
export type ErrorKey = TFuncKey<"error", undefined>
export type LangsKey = TFuncKey<"langs", undefined>
export type NotifyKey = TFuncKey<"notify", undefined>
// export type ValidateKey = TFuncKey<'validate', undefined>

export type OrArray<T> = T | T[];
