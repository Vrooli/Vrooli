import award from "./locales/en/award.json" assert { type: "json" };
import common from "./locales/en/common.json" assert { type: "json" };
import error from "./locales/en/error.json" assert { type: "json" };
import langs from "./locales/en/langs.json" assert { type: "json" };
import notify from "./locales/en/notify.json" assert { type: "json" };
import tasks from "./locales/en/tasks.json" assert { type: "json" };
// const validate = require('./locales/en/validate.json')

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
        tasks,
    },
} as const;

export const i18nConfig = (debug: boolean) => ({
    debug,
    partialBundledLanguages: true,
    defaultNS,
    ns: ["common", "error", "notify", "award", "tasks", "langs"], // 'validate'
    nsSeparator: ":",
    fallbackLng: "en",
    resources,
    backend: {
        loadPath: "./locales/{{lng}}/{{ns}}.json",
    },
});
