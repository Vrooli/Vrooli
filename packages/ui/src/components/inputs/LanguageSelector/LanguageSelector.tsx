import { useCallback, useContext, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { getSiteLanguage, siteLanguages } from 'utils/authentication/session';
import { AllLanguages } from 'utils/display/translationTools';
import { PubSub } from 'utils/pubsub';
import { SessionContext } from 'utils/SessionContext';
import { Selector } from '../Selector/Selector';
import { LanguageSelectorProps } from '../types';

/**
 * Updates the language of the entire app
 */
export function LanguageSelector({
}: LanguageSelectorProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    const [language, setLanguage] = useState(() => getSiteLanguage(session));

    const handleChange = useCallback((newLanguage: string) => {
        setLanguage(newLanguage);
        PubSub.get().publishLanguage(newLanguage);
    }, []);

    return (
        <Selector
            id="languageSelector"
            options={siteLanguages}
            getOptionLabel={(r) => AllLanguages[r]}
            selected={language}
            handleChange={handleChange}
            fullWidth={true}
            inputAriaLabel="Language"
            label={t('Language', { count: 1 })}
        />
    )
}
