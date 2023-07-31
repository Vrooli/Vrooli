import { i18nConfig } from "@local/shared";
import i18n from "i18next";
import { initReactI18next } from "react-i18next";

/** Logs info like missing translation keys */
const debug = import.meta.env.DEV;
/** Displays keys instead of translations. Useful to find missing translations */
export const CI_MODE = false;

i18n.use(initReactI18next).init(i18nConfig(debug, CI_MODE));
if (CI_MODE) {
    i18n.changeLanguage("cimode");
}

export default i18n;
