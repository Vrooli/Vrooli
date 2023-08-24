import { endpointGetNoteVersion, NoteVersion } from "@local/shared";
import { useTheme } from "@mui/material";
import { EllipsisActionButton } from "components/buttons/EllipsisActionButton/EllipsisActionButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { MarkdownInputBase } from "components/inputs/MarkdownInputBase/MarkdownInputBase";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { toDisplay } from "utils/display/pageTools";
import { firstString } from "utils/display/stringTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { NoteViewProps } from "../types";

export const NoteView = ({
    isOpen,
    onClose,
}: NoteViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const display = toDisplay(isOpen);

    const { id, isLoading, object: noteVersion, setObject: setNoteVersion } = useObjectFromUrl<NoteVersion>({
        ...endpointGetNoteVersion,
        objectType: "NoteVersion",
    });

    const availableLanguages = useMemo<string[]>(() => (noteVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [noteVersion?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { description, name, text } = useMemo(() => {
        const { description, name, pages } = getTranslation(noteVersion, [language]);
        return {
            description: description && description.trim().length > 0 ? description : undefined,
            name,
            text: pages?.sort((a, b) => a.pageIndex - b.pageIndex).map(p => p.text).join("\n\n") ?? "",
        };
    }, [language, noteVersion]);

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
                title={firstString(name, t("Note", { count: 1 }))}
                below={availableLanguages.length > 1 && <SelectLanguageMenu
                    currentLanguage={language}
                    handleCurrent={setLanguage}
                    languages={availableLanguages}
                />}
            />
            <>
                <SideActionButtons display={display}>
                    <EllipsisActionButton>
                        <ObjectActionsRow
                            actionData={actionData}
                            object={noteVersion}
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
                            position: "sticky",
                            top: 0,
                        },
                        root: {
                            height: "100%",
                            position: "relative",
                            maxWidth: "800px",
                            margin: "auto",
                            ...(display === "page" ? {
                                marginBottom: 4,
                                borderRadius: { xs: 0, md: 1 },
                                overflow: "overlay",
                            } : {}),
                        },
                        textArea: {
                            borderRadius: 0,
                            resize: "none",
                            height: "100%",
                            overflow: "hidden", // Container handles scrolling
                            background: palette.background.paper,
                            border: "none",
                            ...(display === "page" ? {
                                minHeight: "100vh",
                            } : {}),
                        },
                    }}
                />
            </>
        </>
    );
};
