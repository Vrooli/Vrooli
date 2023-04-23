import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Stack, Typography } from "@mui/material";
import { useCallback, useContext } from "react";
import { getLanguageSubtag, getUserLanguages } from "../../../utils/display/translationTools";
import { SessionContext } from "../../../utils/SessionContext";
import { SelectLanguageMenu } from "../../dialogs/SelectLanguageMenu/SelectLanguageMenu";
export const LanguageInput = ({ currentLanguage, disabled, handleAdd, handleDelete, handleCurrent, languages, zIndex, }) => {
    const session = useContext(SessionContext);
    const selectLanguage = useCallback((language) => {
        if (!languages.some((l) => getLanguageSubtag(l) === getLanguageSubtag(language))) {
            handleAdd(language);
        }
        handleCurrent(language);
    }, [handleAdd, handleCurrent, languages]);
    const deleteLanguage = useCallback((language) => {
        const newList = languages.filter((l) => getLanguageSubtag(l) !== getLanguageSubtag(language));
        if (newList.length === 0) {
            const userLanguages = getUserLanguages(session);
            if (userLanguages.length > 0) {
                handleCurrent(userLanguages[0]);
                handleAdd(userLanguages[0]);
            }
            else {
                handleCurrent("en");
                handleAdd("en");
            }
        }
        else {
            handleCurrent(newList[0]);
        }
        handleDelete(language);
    }, [handleAdd, handleCurrent, handleDelete, session, languages]);
    return (_jsxs(Stack, { direction: "row", spacing: 1, children: [_jsx(SelectLanguageMenu, { currentLanguage: currentLanguage, handleDelete: deleteLanguage, handleCurrent: selectLanguage, isEditing: true, languages: languages, sxs: {
                    root: { marginLeft: 0.5, marginRight: 0.5 },
                }, zIndex: zIndex + 1 }), languages.length > 1 && (_jsxs(Typography, { variant: "body1", sx: {
                    display: "flex",
                    alignItems: "center",
                }, children: ["+", languages.length - 1] }))] }));
};
//# sourceMappingURL=LanguageInput.js.map