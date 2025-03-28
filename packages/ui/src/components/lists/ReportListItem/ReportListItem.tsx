import { ReportStatus } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../../contexts.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { firstString } from "../../../utils/display/stringTools.js";
import { getUserLanguages } from "../../../utils/display/translationTools.js";
import { DateDisplay } from "../../text/DateDisplay.js";
import { ListItemChip, ListItemStyleColor, ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase.js";
import { ReportListItemProps } from "../types.js";

function statusToColor(status: ReportStatus | undefined): ListItemStyleColor {
    if (!status) return "Default";
    switch (status) {
        case ReportStatus.Open:
            return "Yellow";
        case ReportStatus.ClosedFalseReport:
        case ReportStatus.ClosedNonIssue:
            return "Green";
        case ReportStatus.ClosedDeleted:
        case ReportStatus.ClosedHidden:
        case ReportStatus.ClosedSuspended:
            return "Red";
        default: return "Default";
    }
}

export function ReportListItem({
    data,
    ...props
}: ReportListItemProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const languages = getUserLanguages(session);

    const { title } = useMemo(() => {
        const display = getDisplay(data, languages);
        const title = firstString(display.title, data?.reason, t("Report"));
        return {
            title,
        };
    }, [data, languages, t]);

    return (
        <ObjectListItemBase
            {...props}
            data={data}
            objectType="Report"
            titleOverride={title}
            belowTags={
                <Box display="flex" flexDirection="row" alignItems="center" gap={1}>
                    {data?.created_at && (
                        <DateDisplay
                            showIcon={true}
                            timestamp={data.created_at}
                        />
                    )}
                    {data?.status && (
                        <ListItemChip
                            color={statusToColor(data.status)}
                            // label={t(data.status)}
                            label={data.status} //TODO
                        />
                    )}
                    {data?.you?.isOwn && (
                        <ListItemChip
                            color="Blue"
                            label={t("Yours")}
                        />
                    )}
                    {data?.responsesCount !== undefined && data.responsesCount > 0 && (
                        <ListItemChip
                            color="Purple"
                            label={t("Response", { count: data.responsesCount })}
                        />
                    )}
                </Box>
            }
        />
    );
}
