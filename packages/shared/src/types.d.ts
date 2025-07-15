/* c8 ignore start */
import { type TFuncKey, type TFunction } from "i18next";
import "yup";
import type award from "./translations/locales/en/award.json";
import type common from "./translations/locales/en/common.json";
import type error from "./translations/locales/en/error.json";
import type langs from "./translations/locales/en/langs.json";
import type notify from "./translations/locales/en/notify.json";
import type service from "./translations/locales/en/service.json";

declare module "@vrooli/shared";

declare module "i18next" {
    interface CustomTypeOptions {
        defaultNS: "common";
        resources: {
            award: typeof award;
            common: typeof common;
            error: typeof error;
            langs: typeof langs;
            notify: typeof notify;
            service: typeof service;
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
export type TranslationKeyAward = TFuncKey<"award", undefined>
export type TranslationKeyCommon = TFuncKey<"common", undefined>
export type TranslationKeyError = TFuncKey<"error", undefined>
export type TranslationKeyLangs = TFuncKey<"langs", undefined>
export type TranslationKeyNotify = TFuncKey<"notify", undefined>
export type TranslationKeyService = TFuncKey<"service", undefined>

export type TranslationFuncAward = TFunction<"award", undefined, "award">
export type TranslationFuncCommon = TFunction<"common", undefined, "common">
export type TranslationFuncError = TFunction<"error", undefined, "error">
export type TranslationFuncLangs = TFunction<"langs", undefined, "langs">
export type TranslationFuncNotify = TFunction<"notify", undefined, "notify">
export type TranslationFuncService = TFunction<"service", undefined, "service">
export type TranslationFunc = TFunction<"award" | "common" | "error" | "langs" | "notify" | "service", undefined, "award" | "common" | "error" | "langs" | "notify" | "service">

export type OrArray<T> = T | T[];
export type ArrayElement<T> = T extends (infer E)[] ? E : T;
export type DefinedArrayElement<T> = T extends (infer E)[] ? E : NonNullable<T>;
