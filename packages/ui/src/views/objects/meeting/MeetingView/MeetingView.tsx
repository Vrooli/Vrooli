import { endpointsMeeting, Meeting } from "@local/shared";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { TopBar } from "components/navigation/TopBar/TopBar.js";
import { SessionContext } from "contexts";
import { useObjectActions } from "hooks/objectActions.js";
import { useManagedObject } from "hooks/useManagedObject.js";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "utils/display/translationTools.js";
import { MeetingViewProps } from "../types.js";

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
