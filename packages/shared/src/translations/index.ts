import award from "./locales/en/award.json" with { type: "json" };
import common from "./locales/en/common.json" with { type: "json" };
import error from "./locales/en/error.json" with { type: "json" };
import langs from "./locales/en/langs.json" with { type: "json" };
import notify from "./locales/en/notify.json" with { type: "json" };
import service from "./locales/en/service.json" with { type: "json" };
import tasks from "./locales/en/tasks.json" with { type: "json" };
// import validate from "./locales/en/validate.json" with { type: "json" };

// Setup internationization
const defaultNS = "common";
const resources = {
    en: {
        award,
        common,
        // validate,
        error,
        langs,
        notify,
        service,
        tasks,
    },
} as const;

/**
 * Configuration for i18next.
 * @param debug Whether to enable debug mode, which logs to the console when
 * translations are missing.
 * @param appendNS Appends the namespace to the key (e.g. "common:hello") instead 
 * of translating the key directly (e.g. "hello"). Useful for finding strings 
 * that are missing from the translation files.
 */
export function i18nConfig(debug: boolean, appendNS = true) {
    return {
        debug,
        partialBundledLanguages: true,
        defaultNS,
        ns: ["common", "error", "notify", "award", "tasks", "service", "langs"], // 'validate'
        nsSeparator: ":",
        fallbackLng: "en",
        resources,
        backend: {
            loadPath: "./locales/{{lng}}/{{ns}}.json",
        },
        appendNamespaceToCIMode: appendNS,
    };
}

export * from "./translationTools.js";
