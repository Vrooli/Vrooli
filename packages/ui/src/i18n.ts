import { i18nConfig } from '@shared/translations';
import i18n from "i18next";
import { initReactI18next } from "react-i18next";

const debug = import.meta.env.DEV;

i18n.use(initReactI18next).init(i18nConfig(debug));

export default i18n;