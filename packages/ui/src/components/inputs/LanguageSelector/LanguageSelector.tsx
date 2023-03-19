import { Formik } from 'formik';
import { useContext } from 'react';
import { useTranslation } from 'react-i18next';
import { getSiteLanguage, siteLanguages } from 'utils/authentication/session';
import { AllLanguages } from 'utils/display/translationTools';
import { PubSub } from 'utils/pubsub';
import { SessionContext } from 'utils/SessionContext';
import { Selector } from '../Selector/Selector';

/**
 * Updates the language of the entire app
 */
export function LanguageSelector() {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    return (
        <Formik
            enableReinitialize={true}
            initialValues={{ language: getSiteLanguage(session) }}
            onSubmit={() => { }} // no-op
        >
            <Selector
                name="language"
                options={siteLanguages}
                getOptionLabel={(r) => AllLanguages[r]}
                fullWidth={true}
                inputAriaLabel="Language"
                label={t('Language', { count: 1 })}
                onChange={(newLang) => { newLang && PubSub.get().publishLanguage(newLang) }}
            />
        </Formik>
    )
}
