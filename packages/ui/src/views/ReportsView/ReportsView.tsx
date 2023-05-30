import { useQuery } from "@apollo/client";
import { getLastUrlPart, Report, ReportSearchInput, ReportSearchResult } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { reportFindMany } from "api/generated/endpoints/report_findMany";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Wrap } from "types";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { ReportsViewProps } from "../types";

/**
 * Maps object types to the correct id fields
 */
const objectTypeToIdField = {
    "Comment": "commentId",
    "Organization": "organizationId",
    "Project": "projectId",
    "Routine": "routineId",
    "Standard": "standardId",
    "Tag": "tagId",
    "User": "userId",
};

export const ReportsView = ({
    display = "page",
    onClose,
}: ReportsViewProps): JSX.Element => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const objectType = useMemo(() => getLastUrlPart(1), []);

    const { data } = useQuery<Wrap<ReportSearchResult, "reports">, Wrap<ReportSearchInput, "input">>(
        reportFindMany,
        { variables: { input: { [objectTypeToIdField[objectType]]: id } } },
    );
    const reports = useMemo<Report[]>(() => {
        if (!data) return [];
        return data.reports.edges.map(edge => edge.node);
    }, [data]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                titleData={{
                    titleKey: "Reports",
                    helpKey: "ReportsHelp",
                }}
            />
            {reports.map((report, i) => {
                return <Box
                    key={i}
                    sx={{
                        background: palette.background.paper,
                        color: palette.background.textPrimary,
                        borderRadius: "16px",
                        boxShadow: 12,
                        margin: "16px 16px 0 16px",
                        padding: "1rem",
                    }}
                >
                    <p style={{ margin: "0" }}>
                        <b>{t("Reason")}:</b> {report.reason}
                    </p>
                    <p style={{ margin: "1rem 0 0 0" }}>
                        <b>{t("Details")}:</b>  {report.details}
                    </p>
                </Box>;
            })}
        </>
    );
};
