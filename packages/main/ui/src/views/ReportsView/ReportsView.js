import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { useQuery } from "@apollo/client";
import { Box, useTheme } from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { reportFindMany } from "../../api/generated/endpoints/report_findMany";
import { TopBar } from "../../components/navigation/TopBar/TopBar";
import { parseSingleItemUrl } from "../../utils/navigation/urlTools";
import { getLastUrlPart } from "../../utils/route";
const objectTypeToIdField = {
    "Comment": "commentId",
    "Organization": "organizationId",
    "Project": "projectId",
    "Routine": "routineId",
    "Standard": "standardId",
    "Tag": "tagId",
    "User": "userId",
};
export const ReportsView = ({ display = "page", }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const objectType = useMemo(() => getLastUrlPart(1), []);
    const { data } = useQuery(reportFindMany, { variables: { input: { [objectTypeToIdField[objectType]]: id } } });
    const reports = useMemo(() => {
        if (!data)
            return [];
        return data.reports.edges.map(edge => edge.node);
    }, [data]);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Reports",
                    helpKey: "ReportsHelp",
                } }), reports.map((report, i) => {
                return _jsxs(Box, { sx: {
                        background: palette.background.paper,
                        color: palette.background.textPrimary,
                        borderRadius: "16px",
                        boxShadow: 12,
                        margin: "16px 16px 0 16px",
                        padding: "1rem",
                    }, children: [_jsxs("p", { style: { margin: "0" }, children: [_jsxs("b", { children: [t("Reason"), ":"] }), " ", report.reason] }), _jsxs("p", { style: { margin: "1rem 0 0 0" }, children: [_jsxs("b", { children: [t("Details"), ":"] }), "  ", report.details] })] }, i);
            })] }));
};
//# sourceMappingURL=ReportsView.js.map