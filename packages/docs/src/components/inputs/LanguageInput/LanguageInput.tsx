/**
 * Handles the state management for adding/updating text in multiple languages.
 */
import { Stack, Typography } from '@mui/material';
import { LanguageInputProps } from '../types';
import { useCallback } from 'react';
import { SelectLanguageDialog } from 'components/dialogs';
import { AllLanguages, getUserLanguages } from 'utils';

export const LanguageInput = ({
    currentLanguage,
    disabled,
    handleAdd,
    handleDelete,
    handleCurrent,
    selectedLanguages,
    session,
    zIndex,
}: LanguageInputProps) => {
    /**
     * When a language is selected, call handleCurrent and add it to the list of selected languages.
     * Do not delete if already selected, as that is done separately.
     */
    const selectLanguage = useCallback((language: string) => {
        if (selectedLanguages.indexOf(language) === -1) {
            handleAdd(language);
        }
        handleCurrent(language);
    }, [handleAdd, handleCurrent, selectedLanguages]);

    /**
     * Remove a language from the list of selected languages.
     * If the language is the current language, call handleCurrent with
     * one of the other selected languages, or the first user language if none are selected.
     */
    const deleteLanguage = useCallback((language: string) => {
        const newList = selectedLanguages.filter(l => l !== language);
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
            handleCurrent(newList[0]);
        }
        handleDelete(language);
    }, [handleAdd, handleCurrent, handleDelete, selectedLanguages, session]);

    return (
        <Stack direction="row" spacing={1}>
            <SelectLanguageDialog
                availableLanguages={Object.keys(AllLanguages)}
                currentLanguage={currentLanguage}
                canDropdownOpen={true}
                handleDelete={deleteLanguage}
                handleCurrent={selectLanguage}
                isEditing={true}
                selectedLanguages={selectedLanguages}
                session={session}
                sxs={{
                    root: { marginLeft: 0.5, marginRight: 0.5 },
                }}
                zIndex={zIndex + 1}
            />
            {
                selectedLanguages.length > 1 && (
                    <Typography variant="body1" sx={{
                        marginTop: 'auto',
                        marginBottom: 'auto',
                        marginLeft: 0,
                    }}>
                        +{selectedLanguages.length - 1}
                    </Typography>
                )
            }
        </Stack>
    )
}