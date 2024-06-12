import { SessionContext } from "contexts/SessionContext";
import { useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { getSiteLanguage, siteLanguages } from "utils/authentication/session";
import { AllLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { SelectorBase } from "../Selector/Selector";

/**
 * Updates the language of the entire app
 */
export function LanguageSelector() {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    const [language, setLanguage] = useState(getSiteLanguage(session));
    const handleLanguageChange = (newLang: string) => {
        setLanguage(newLang);
        PubSub.get().publish("language", newLang);
    };

    const handleRequestLanguage = () => {
        // Open link to new issue on github with title "Support for language <enter_language_here>"
        const PROJECT_URL = "https://github.com/Vrooli/Vrooli";
        const ISSUE_URL = `${PROJECT_URL}/issues/new?assignees=&labels=✨+enhancement&projects=&template=--feature-request.md&title=Support+for+language+%3Center_language_here%3E`;
        window.open(ISSUE_URL, "_blank");
    };

    return (
        <SelectorBase
            name="language"
            options={siteLanguages}
            getOptionLabel={(r) => AllLanguages[r]}
            fullWidth={true}
            inputAriaLabel="Language"
            label={t("Language", { count: 1 })}
            value={language}
            onChange={handleLanguageChange}
            addOption={{
                label: t("RequestLanguage"),
                onSelect: handleRequestLanguage,
            }}
        />
    );
}
