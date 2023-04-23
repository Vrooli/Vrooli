import { jsx as _jsx } from "react/jsx-runtime";
import { Formik } from "formik";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { getSiteLanguage, siteLanguages } from "../../../utils/authentication/session";
import { AllLanguages } from "../../../utils/display/translationTools";
import { PubSub } from "../../../utils/pubsub";
import { SessionContext } from "../../../utils/SessionContext";
import { Selector } from "../Selector/Selector";
export function LanguageSelector() {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    return (_jsx(Formik, { enableReinitialize: true, initialValues: { language: getSiteLanguage(session) }, onSubmit: () => { }, children: _jsx(Selector, { name: "language", options: siteLanguages, getOptionLabel: (r) => AllLanguages[r], fullWidth: true, inputAriaLabel: "Language", label: t("Language", { count: 1 }), onChange: (newLang) => { newLang && PubSub.get().publishLanguage(newLang); } }) }));
}
//# sourceMappingURL=LanguageSelector.js.map