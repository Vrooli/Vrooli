import { useCallback, useState } from 'react';
import { LanguageSelectorProps } from '../types';
import { useTranslation } from 'react-i18next';
import { AllLanguages, PubSub } from 'utils';
import { Selector } from '../Selector/Selector';
import { getSiteLanguage, siteLanguages } from 'utils/authentication';

/**
 * Updates the language of the entire app
 */
export function LanguageSelector({
    session,
}: LanguageSelectorProps) {
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
