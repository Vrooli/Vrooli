import { LanguageIcon } from "icons/common.js";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SessionService, siteLanguages } from "utils/authentication/session.js";
import { AllLanguages } from "utils/display/translationTools.js";
import { PubSub } from "utils/pubsub.js";
import { SessionContext } from "../../../contexts.js";
import { SelectorBase } from "../Selector/Selector.js";

function handleRequestLanguage() {
    // Open link to new issue on github with title "Support for language <enter_language_here>"
    const PROJECT_URL = "https://github.com/Vrooli/Vrooli";
    const ISSUE_URL = `${PROJECT_URL}/issues/new?assignees=&labels=✨+enhancement&projects=&template=--feature-request.md&title=Support+for+language+%3Center_language_here%3E`;
    window.open(ISSUE_URL, "_blank");
}

function getLanguageOptionLabel(lang: string) {
    return AllLanguages[lang] ?? lang;
}

function getDisplayIcon() {
    return <LanguageIcon />;
}

/**
 * Updates the language of the entire app
 */
export function LanguageSelector() {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    const [language, setLanguage] = useState(SessionService.getSiteLanguage(session));
    const handleLanguageChange = useCallback(function handleLanguageChangeCallback(newLang: string) {
        setLanguage(newLang);
        // Let the app know the language has changed
        PubSub.get().publish("language", newLang);
    }, []);

    const addOption = useMemo(function addOptionMemo() {
        return {
            label: t("RequestLanguage"),
            onSelect: handleRequestLanguage,
        };
    }, [t]);

    return (
        <SelectorBase
            name="language"
            options={siteLanguages}
            getDisplayIcon={getDisplayIcon}
            getOptionLabel={getLanguageOptionLabel}
            fullWidth={true}
            inputAriaLabel="Language"
            label={t("Language", { count: 1 })}
            value={language}
            onChange={handleLanguageChange}
            addOption={addOption}
        />
    );
}
