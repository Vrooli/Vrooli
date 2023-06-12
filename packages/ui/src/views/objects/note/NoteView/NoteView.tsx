import { endpointGetNoteVersion, NoteVersion, useLocation } from "@local/shared";
import { useTheme } from "@mui/material";
import { EllipsisActionButton } from "components/buttons/EllipsisActionButton/EllipsisActionButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { MarkdownInputBase } from "components/inputs/MarkdownInputBase/MarkdownInputBase";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { firstString } from "utils/display/stringTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { SessionContext } from "utils/SessionContext";
import { NoteViewProps } from "../types";

export const NoteView = ({
    display = "page",
    onClose,
    partialData,
    zIndex = 200,
}: NoteViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { id, isLoading, object: noteVersion, setObject: setNoteVersion } = useObjectFromUrl<NoteVersion>({
        ...endpointGetNoteVersion,
        partialData,
    });

    const availableLanguages = useMemo<string[]>(() => (noteVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [noteVersion?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
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

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={firstString(name, t("Note"))}
                below={availableLanguages.length > 1 && <SelectLanguageMenu
                    currentLanguage={language}
                    handleCurrent={setLanguage}
                    languages={availableLanguages}
                    zIndex={zIndex}
                />}
            />
            <>
                <SideActionButtons display={display} zIndex={zIndex + 1}>
                    <EllipsisActionButton>
                        <ObjectActionsRow
                            actionData={actionData}
                            object={noteVersion}
                            zIndex={zIndex}
                        />
                    </EllipsisActionButton>
                </SideActionButtons>
                <MarkdownInputBase
                    disabled={true}
                    minRows={3}
                    name="text"
                    onChange={() => { }}
                    value={text}
                    sxs={{
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
                    }}
                    zIndex={zIndex}
                />
            </>
        </>
    );
};
