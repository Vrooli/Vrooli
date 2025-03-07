import { getObjectSlug, getObjectUrlBase } from "@local/shared";
import { IconButton, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { ReportIcon } from "icons/common.js";
import { useCallback, useMemo } from "react";
import { useLocation } from "route";
import { ReportsLinkProps } from "../types.js";

const CountLabel = styled(Typography)(({ theme }) => ({
    marginLeft: theme.spacing(0.5),
    color: theme.palette.background.textPrimary,
}));

/**
 * Renders a link that says 'Reports (x)' where x is the number of reports.
 * When clicked, navigates to href.
 */
export function ReportsLink({
    object,
}: ReportsLinkProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    // We set href and onClick so users can open in new tab, while also supporting single-page app navigation
    const link = useMemo<string>(() => object ? `/reports${getObjectUrlBase(object)}/${getObjectSlug(object, true)}` : "", [object]);

    const handleClick = useCallback(function handleClickCallback(ev: React.MouseEvent<HTMLAnchorElement>) {
        setLocation(link);
        // Prevent default so we don't use href
        ev.preventDefault();
    }, [link, setLocation]);

    if (!object?.reportsCount || object.reportsCount <= 0) return null;
    return (
        <Tooltip title="Press to view and repond to reports.">
            <IconButton
                href={link}
                onClick={handleClick}
            >
                <ReportIcon fill={palette.background.textPrimary} />
                <CountLabel variant="body1">({object.reportsCount})</CountLabel>
            </IconButton>
        </Tooltip>
    );
}
