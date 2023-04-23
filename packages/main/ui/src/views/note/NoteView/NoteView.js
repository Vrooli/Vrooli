import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { useTheme } from "@mui/material";
import { useContext, useEffect, useMemo, useState } from "react";
import { noteVersionFindOne } from "../../../api/generated/endpoints/noteVersion_findOne";
import { EllipsisActionButton } from "../../../components/buttons/EllipsisActionButton/EllipsisActionButton";
import { SideActionButtons } from "../../../components/buttons/SideActionButtons/SideActionButtons";
import { MarkdownInputBase } from "../../../components/inputs/MarkdownInputBase/MarkdownInputBase";
import { ObjectActionsRow } from "../../../components/lists/ObjectActionsRow/ObjectActionsRow";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "../../../utils/display/translationTools";
import { useObjectActions } from "../../../utils/hooks/useObjectActions";
import { useObjectFromUrl } from "../../../utils/hooks/useObjectFromUrl";
import { useLocation } from "../../../utils/route";
import { SessionContext } from "../../../utils/SessionContext";
export const NoteView = ({ display = "page", partialData, zIndex = 200, }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { id, isLoading, object: noteVersion, setObject: setNoteVersion } = useObjectFromUrl({
        query: noteVersionFindOne,
        partialData,
    });
    const availableLanguages = useMemo(() => (noteVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [noteVersion?.translations]);
    const [language, setLanguage] = useState(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0)
            return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);
    const { description, name, text } = useMemo(() => {
        const { description, name, text } = getTranslation(noteVersion ?? partialData, [language]);
        return {
            description: description && description.trim().length > 0 ? description : undefined,
            name,
            text: text ?? "",
        };
    }, [language, noteVersion, partialData]);
    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);
    const actionData = useObjectActions({
        object: noteVersion,
        objectType: "NoteVersion",
        setLocation,
        setObject: setNoteVersion,
    });
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Note",
                } }), _jsxs(_Fragment, { children: [_jsx(SideActionButtons, { display: display, zIndex: zIndex + 1, children: _jsx(EllipsisActionButton, { children: _jsx(ObjectActionsRow, { actionData: actionData, object: noteVersion, zIndex: zIndex }) }) }), _jsx(MarkdownInputBase, { disabled: true, minRows: 3, name: "text", onChange: () => { }, value: text, sxs: {
                            bar: {
                                borderRadius: 0,
                                background: palette.primary.main,
                            },
                            textArea: {
                                borderRadius: 0,
                                resize: "none",
                                minHeight: "100vh",
                                background: palette.background.paper,
                            },
                        } })] })] }));
};
//# sourceMappingURL=NoteView.js.map