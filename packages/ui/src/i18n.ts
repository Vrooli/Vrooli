import i18n from "i18next";
import { initReactI18next  } from "react-i18next";
import { i18nConfig } from '@shared/translations';

const debug = process.env.NODE_ENV !== 'production';

i18n.use(initReactI18next).init(i18nConfig(debug));

export default i18n;