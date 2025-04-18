import { endpointsMeeting, Meeting } from "@local/shared";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SelectLanguageMenu } from "../../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { useObjectActions } from "../../../hooks/objectActions.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useLocation } from "../../../route/router.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { firstString } from "../../../utils/display/stringTools.js";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "../../../utils/display/translationTools.js";
import { MeetingViewProps } from "./types.js";

export function MeetingView({
    display,
    onClose,
}: MeetingViewProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    const { object: existing, isLoading, setObject: setMeeting } = useManagedObject<Meeting>({
        ...endpointsMeeting.findOne,
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
                />}
            />
            <>
                {/* TODO */}
            </>
        </>
    );
}
