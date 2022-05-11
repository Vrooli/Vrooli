/**
 * Handles the state management for adding/updating text in multiple languages.
 */
import { Box, IconButton, Typography } from '@mui/material';
import { LanguageInputProps } from '../types';
import {
    Add as AddIcon,
} from '@mui/icons-material';
import { useCallback, useMemo } from 'react';
import { SelectLanguageDialog } from 'components/dialogs';
import { AllLanguages, getUserLanguages } from 'utils';

export const LanguageInput = ({
    currentLanguage,
    handleAdd,
    handleChange,
    handleDelete,
    handleSelect,
    languages,
    session,
}: LanguageInputProps) => {
    console.log('in languages input', languages);
    const canAdd = useMemo(() => Object.keys(AllLanguages).filter(l => languages.indexOf(l) === -1).length > 0, [languages]);
    const handleAddButtonClick = useCallback(() => {
        // Try to default to first user language that isn't in list already
        const userLanguages = getUserLanguages(session).filter(l => languages.indexOf(l) === -1);
        if (userLanguages.length > 0) {
            handleAdd(userLanguages[0]);
        }
        // Otherwise, default to first language in all languages list, which is not already in list
        else {
            const remainingLanguages = Object.keys(AllLanguages).filter(l => languages.indexOf(l) === -1);
            if (remainingLanguages.length > 0) {
                handleAdd(remainingLanguages[0]);
            }
        }
    }, [handleAdd, languages, session]);

    /**
     * Each language is displayed as a as a SelectLanguageDialog
     */
    const tags = useMemo(() => {
        let tagList = languages.map(l => {
            // Filter out other languages in list so they can't be selected again
            const availableLanguages: string[] = Object.keys(AllLanguages).filter(l2 => l2 === l || languages.indexOf(l2) === -1);
            return (
                <SelectLanguageDialog
                    availableLanguages={availableLanguages}
                    canDelete={!currentLanguage || (l === currentLanguage && languages.length > 1)}
                    canDropdownOpen={!currentLanguage || l === currentLanguage}
                    color={!!currentLanguage && l === currentLanguage ? '#3790a7' : 'default'}
                    handleDelete={() => { handleDelete(l) }}
                    handleSelect={(newLanguage: string) => { handleChange(l, newLanguage) }}
                    key={l}
                    language={l}
                    onClick={() => { l !== currentLanguage && handleSelect(l) }}
                    session={session}
                    sxs={{
                        root: { marginLeft: 0.5, marginRight: 0.5 },
                    }}
                />
            )
        });
        return tagList;
    }, [currentLanguage, handleChange, handleDelete, handleSelect, languages, session]);

    return (
        <Box sx={{
            alignItems: 'center',
            display: 'flex',
            flexWrap: 'wrap',
            rowGap: 1,
        }}>
            <Typography variant="h6">Languages:</Typography>
            {tags}
            {canAdd && (
                <IconButton
                    onClick={handleAddButtonClick}
                    sx={{
                        width: 'fit-content',
                        padding: '0',
                        display: 'flex',
                        alignItems: 'center',
                        backgroundColor: '#6daf72',
                        color: 'white',
                        borderRadius: '100%',
                        '&:hover': {
                            backgroundColor: '#6daf72',
                            filter: `brightness(110%)`,
                            transition: 'filter 0.2s',
                        },
                    }}
                >
                    <AddIcon />
                </IconButton>
            )}
        </Box>
    )
}