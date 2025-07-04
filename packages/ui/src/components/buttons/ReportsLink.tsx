import { IconButton } from "./IconButton.js";
import { Tooltip } from "../Tooltip/Tooltip.js";
import Typography from "@mui/material/Typography";
import { styled, useTheme } from "@mui/material/styles";
import { getObjectSlug, getObjectUrlBase } from "@vrooli/shared";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { type ReportsLinkProps } from "./types.js";

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
    const { t } = useTranslation();
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
                variant="transparent"
                size="md"
                aria-label={t("Report", { count: object.reportsCount })}
                onClick={handleClick}
            >
                <IconCommon
                    decorative
                    fill={palette.background.textPrimary}
                    name="Report"
                />
                <CountLabel variant="body1">({object.reportsCount})</CountLabel>
            </IconButton>
        </Tooltip>
    );
}
