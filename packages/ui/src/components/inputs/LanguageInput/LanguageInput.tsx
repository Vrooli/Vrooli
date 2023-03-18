/**
 * Handles the state management for adding/updating text in multiple languages.
 */
import { Stack, Typography } from '@mui/material';
import { SelectLanguageMenu } from 'components/dialogs/SelectLanguageMenu/SelectLanguageMenu';
import { useCallback, useContext } from 'react';
import { getLanguageSubtag, getUserLanguages } from 'utils/display/translationTools';
import { SessionContext } from 'utils/SessionContext';
import { LanguageInputProps } from '../types';

export const LanguageInput = ({
    currentLanguage,
    disabled,
    handleAdd,
    handleDelete,
    handleCurrent,
    translations,
    zIndex,
}: LanguageInputProps) => {
    const session = useContext(SessionContext);

    const selectLanguage = useCallback((language: string) => {
        // If language is not in translations, add it
        if (!translations.some((translation) => getLanguageSubtag(translation.language) === getLanguageSubtag(language))) {
            handleAdd(language);
        }
        // Select language
        handleCurrent(language);
    }, [handleAdd, handleCurrent, translations]);

    /**
     * Remove a language from the list of selected languages.
     * If the language is the current language, call handleCurrent with
     * one of the other selected languages, or the first user language if none are selected.
     */
    const deleteLanguage = useCallback((language: string) => {
        const newList = translations.filter((translation) => getLanguageSubtag(translation.language) !== getLanguageSubtag(language));
        // If deleting this language makes the list empty, add the first user language
        if (newList.length === 0) {
            const userLanguages = getUserLanguages(session);
            if (userLanguages.length > 0) {
                handleCurrent(userLanguages[0]);
                handleAdd(userLanguages[0]);
            } else {
                handleCurrent('en');
                handleAdd('en');
            }
        }
        // Otherwise, select the first language in the list
        else {
            handleCurrent(newList[0].language);
        }
        handleDelete(language);
    }, [handleAdd, handleCurrent, handleDelete, session, translations]);

    return (
        <Stack direction="row" spacing={1}>
            <SelectLanguageMenu
                currentLanguage={currentLanguage}
                handleDelete={deleteLanguage}
                handleCurrent={selectLanguage}
                isEditing={true}
                sxs={{
                    root: { marginLeft: 0.5, marginRight: 0.5 },
                }}
                translations={translations}
                zIndex={zIndex + 1}
            />
            {/* Display how many translations there are, besides currently selected */}
            {
                translations.length > 1 && (
                    <Typography variant="body1" sx={{
                        display: 'flex',
                        alignItems: 'center',
                    }}>
                        +{translations.length - 1}
                    </Typography>
                )
            }
        </Stack>
    )
}