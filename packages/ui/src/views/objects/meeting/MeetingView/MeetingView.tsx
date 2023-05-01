import { FindByIdInput, Meeting, useLocation } from "@local/shared";
import { useTheme } from "@mui/material";
import { meetingFindOne } from "api/generated/endpoints/meeting_findOne";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { MeetingViewProps } from "../types";

export const MeetingView = ({
    display = "page",
    partialData,
    zIndex = 200,
}: MeetingViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { object: existing, isLoading, setObject: setMeeting } = useObjectFromUrl<Meeting, FindByIdInput>({
        query: meetingFindOne,
        partialData,
    });

    const { name } = useMemo(() => ({ name: getDisplay(existing).title ?? "" }), [existing]);

    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);

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
                onClose={() => { }}
                titleData={{
                    titleKey: "Meeting",
                    titleVariables: { count: 1 },
                }}
            />
            <>
                {/* TODO */}
            </>
        </>
    );
};
