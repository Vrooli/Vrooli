import { i18nConfig } from "@vrooli/shared";
import i18n from "i18next";
import { initReactI18next } from "react-i18next";

/** Logs info like missing translation keys */
const debug = process.env.DEV as unknown as boolean;
/** Displays keys instead of translations. Useful to find missing translations */
export const CI_MODE = false;

// Configure i18next with initImmediate: false for Storybook to prevent backend warnings
// Also disable debug mode in Storybook to prevent console spam
const isStorybook = typeof window !== "undefined" && window.location.href.includes("storybook");
const config = i18nConfig(isStorybook ? false : debug, CI_MODE);
i18n.use(initReactI18next).init({
    ...config,
    initImmediate: false, // Prevents backend connector warnings in Storybook
});

if (CI_MODE) {
    i18n.changeLanguage("cimode");
}

export default i18n;
