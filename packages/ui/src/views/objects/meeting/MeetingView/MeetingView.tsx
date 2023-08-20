import { endpointGetMeeting, Meeting } from "@local/shared";
import { useTheme } from "@mui/material";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getDisplay } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { firstString } from "utils/display/stringTools";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "utils/display/translationTools";
import { MeetingViewProps } from "../types";

export const MeetingView = ({
    isOpen,
    onClose,
    zIndex,
}: MeetingViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const display = toDisplay(isOpen);

    const { object: existing, isLoading, setObject: setMeeting } = useObjectFromUrl<Meeting>({
        ...endpointGetMeeting,
        objectType: "Meeting",
    });

    const availableLanguages = useMemo<string[]>(() => (existing?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [existing?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { name } = useMemo(() => ({ name: getDisplay(existing, [language]).title ?? "" }), [existing, language]);

    const actionData = useObjectActions({
        object: existing,
        objectType: "Meeting",
        setLocation,
        setObject: setMeeting,
    });

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={firstString(name, t("Meeting", { count: 1 }))}
                below={availableLanguages.length > 1 && <SelectLanguageMenu
                    currentLanguage={language}
                    handleCurrent={setLanguage}
                    languages={availableLanguages}
                    zIndex={zIndex}
                />}
                zIndex={zIndex}
            />
            <>
                {/* TODO */}
            </>
        </>
    );
};
