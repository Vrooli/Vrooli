import { IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { ReportIcon } from "@shared/icons";
import { useLocation } from "@shared/route";
import { useCallback, useMemo } from "react";
import { getObjectSlug, getObjectUrlBase } from "utils";
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
    const link = useMemo<string>(() => object ? `${getObjectUrlBase(object)}/reports/${getObjectSlug(object)}` : '', [object]);
    const onClick = useCallback((e: any) => { 
        setLocation(link); 
        // Prevent default so we don't use href
        e.preventDefault();
    }, [link, setLocation]);

    if (!object?.reportsCount || object.reportsCount <= 0) return null;
    return (
        <Tooltip title="Press to view and repond to reports.">
            <IconButton
                href={link}
                onClick={onClick}
            >
                <ReportIcon fill={palette.background.textPrimary} />
                <Typography variant="body1" sx={{ ml: 0.5, color: palette.background.textPrimary }}>({object.reportsCount})</Typography>
            </IconButton>
        </Tooltip>
    )
}