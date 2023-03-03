import { ListItem, ListItemButton, ListItemText, Stack, Typography, useTheme } from "@mui/material";
import { useCallback, useMemo } from "react";
import { multiLineEllipsis } from "styles";
import { getTranslation, getUserLanguages, useTopBar } from "utils";
import { CalendarViewProps } from "../types";
import { HelpButton } from "components/buttons";
import { OpenInNewIcon } from "@shared/icons";
import { Node, NodeLink, NodeType } from "@shared/consts";
import { useTranslation } from "react-i18next";

export const CalendarView = ({
    display = 'page',
    session,
}: CalendarViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);

    // const [eventsData, { data: pageData, loading, error }] = useCustomlazyQuery<QueryResult, QueryVariables>(query, {
    //     variables: ({
    //         after: after.current,
    //         take,
    //         sortBy,
    //         searchString,
    //         createdTimeFrame: (timeFrame && Object.keys(timeFrame).length > 0) ? {
    //             after: timeFrame.after?.toISOString(),
    //             before: timeFrame.before?.toISOString(),
    //         } : undefined,
    //         ...where,
    //         ...advancedSearchParams
    //     } as any),
    //     errorPolicy: 'all',
    // });

    const TopBar = useTopBar({
        display,
        session,
        titleData: {
            titleKey: 'Calendar',
        },
    })

    return (
        <>
            {TopBar}
        </>
    )
}