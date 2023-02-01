import { useQuery } from "@apollo/client";
import { useMemo } from "react";
import { Box, useTheme } from "@mui/material";
import { getLastUrlPart, parseSingleItemUrl } from "utils";
import { PageTitle } from "components/text";
import { ReportsViewPageProps } from "pages/view/types";
import { Report, ReportSearchInput, ReportSearchResult } from "@shared/consts";
import { Wrap } from "types";
import { reportFindMany } from "api/generated/endpoints/report";

/**
 * Maps object types to the correct id fields
 */
const objectTypeToIdField = {
    'Comment': 'commentId',
    'Organization': 'organizationId',
    'Project': 'projectId',
    'Routine': 'routineId',
    'Standard': 'standardId',
    'Tag': 'tagId',
    'User': 'userId',
}

export const ReportsView = ({
    session
}: ReportsViewPageProps): JSX.Element => {
    const { palette } = useTheme();
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const objectType = useMemo(() => getLastUrlPart(1), []);

    const { data } = useQuery<Wrap<ReportSearchResult, 'reports'>, Wrap<ReportSearchInput, 'input'>>(
        reportFindMany,
        { variables: { input: { [objectTypeToIdField[objectType]]: id } } },
    );
    const reports = useMemo<Report[]>(() => {
        if (!data) return []
        return data.reports.edges.map(edge => edge.node);
    }, [data]);

    return (
        <>
            <PageTitle titleKey='Reports' helpKey='ReportsHelp' session={session} />
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
                        <b>Reason:</b> {report.reason}
                    </p>
                    <p style={{ margin: "1rem 0 0 0" }}>
                        <b>Details:</b>  {report.details}
                    </p>
                </Box>
            })}
        </>
    )
}