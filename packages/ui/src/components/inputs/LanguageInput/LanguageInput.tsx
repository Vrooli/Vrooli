/**
 * Handles the state management for adding/updating text in multiple languages.
 */
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useCallback, useContext, useMemo } from "react";
import { SessionContext } from "../../../contexts/session.js";
import { getLanguageSubtag, getUserLanguages } from "../../../utils/display/translationTools.js";
import { SelectLanguageMenu } from "../../dialogs/SelectLanguageMenu/SelectLanguageMenu.js";
import { type LanguageInputProps } from "../types.js";

export function LanguageInput({
    currentLanguage,
    flexDirection,
    handleAdd,
    handleDelete,
    handleCurrent,
    languages,
}: LanguageInputProps) {
    const session = useContext(SessionContext);
    // TODO for morning: improve look of this component, fix bugs with translations when creating/updating, then fix other bugs with creating/updating

    const selectLanguage = useCallback((language: string) => {
        // If language is not in languages, add it
        if (!languages.some((l) => getLanguageSubtag(l) === getLanguageSubtag(language))) {
            handleAdd(language);
        }
        // Select language
        handleCurrent(language);
    }, [handleAdd, handleCurrent, languages]);

    /**
     * Remove a language from the list of selected languages.
     * If the language is the current language, call handleCurrent with
     * one of the other selected languages, or the first user language if none are selected.
     */
    const deleteLanguage = useCallback((language: string) => {
        const newList = languages.filter((l) => getLanguageSubtag(l) !== getLanguageSubtag(language));
        // If deleting this language makes the list empty, add the first user language
        if (newList.length === 0) {
            const userLanguages = getUserLanguages(session);
            if (userLanguages.length > 0) {
                handleCurrent(userLanguages[0]);
                handleAdd(userLanguages[0]);
            } else {
                handleCurrent("en");
                handleAdd("en");
            }
        }
        // Otherwise, select the first language in the list
        else {
            handleCurrent(newList[0]);
        }
        handleDelete(language);
    }, [handleAdd, handleCurrent, handleDelete, session, languages]);

    const selectLanguageMenuSxs = useMemo(() => ({
        root: { marginLeft: 0.5, marginRight: 0.5 },
    }), []);

    const translationCountSx = useMemo(() => ({
        display: "flex",
        alignItems: "center",
    }), []);

    return (
        <Stack direction="row" spacing={1} flexDirection={flexDirection || "row"} data-testid="language-input">
            <SelectLanguageMenu
                currentLanguage={currentLanguage}
                handleDelete={deleteLanguage}
                handleCurrent={selectLanguage}
                isEditing={true}
                languages={languages}
                sxs={selectLanguageMenuSxs}
            />
            {/* Display how many translations there are, besides currently selected */}
            {
                languages.length > 1 && (
                    <Typography 
                        variant="body1" 
                        color="text.secondary" 
                        data-testid="translation-count"
                        sx={translationCountSx}
                    >
                        +{languages.length - 1}
                    </Typography>
                )
            }
        </Stack>
    );
}
