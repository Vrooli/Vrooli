import { i18nConfig } from "@local/translations";
import i18n from "i18next";
import { initReactI18next } from "react-i18next";
const debug = process.env.NODE_ENV === "development";
i18n.use(initReactI18next).init(i18nConfig(debug));
export default i18n;
//# sourceMappingURL=i18n.js.map