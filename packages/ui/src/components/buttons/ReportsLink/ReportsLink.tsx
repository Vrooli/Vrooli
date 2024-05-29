import { getObjectSlug, getObjectUrlBase } from "@local/shared";
import { IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { ReportIcon } from "icons";
import { useMemo } from "react";
import { useLocation } from "route";
import { ReportsLinkProps } from "../types";

/**
 * Renders a link that says 'Reports (x)' where x is the number of reports.
 * When clicked, navigates to href.
 */
export const ReportsLink = ({
    object,
}: ReportsLinkProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    // We set href and onClick so users can open in new tab, while also supporting single-page app navigation
    const link = useMemo<string>(() => object ? `${getObjectUrlBase(object)}/reports/${getObjectSlug(object)}` : "", [object]);

    if (!object?.reportsCount || object.reportsCount <= 0) return null;
    return (
        <Tooltip title="Press to view and repond to reports.">
            <IconButton
                href={link}
                onClick={(ev) => {
                    setLocation(link);
                    // Prevent default so we don't use href
                    ev.preventDefault();
                }}
            >
                <ReportIcon fill={palette.background.textPrimary} />
                <Typography variant="body1" sx={{ ml: 0.5, color: palette.background.textPrimary }}>({object.reportsCount})</Typography>
            </IconButton>
        </Tooltip>
    );
};
