import { Box, Typography, styled, useTheme } from "@mui/material";
import { ReportIcon } from "icons";
import { useCallback, useMemo } from "react";
import { Link, useLocation } from "route";
import { multiLineEllipsis } from "styles";
import { getObjectReportUrl } from "utils/navigation/openObject";
import { ReportsButtonProps } from "../types";

const OuterBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    gap: theme.spacing(0.5),
    pointerEvents: "none",
}));

const CountLabel = styled(Typography)(({ theme }) => ({
    ...multiLineEllipsis(1),
    pointerEvents: "none",
    color: theme.palette.background.textSecondary,
}));

export const ReportsButton = ({
    reportsCount = 0,
    object,
}: ReportsButtonProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const link = useMemo(() => object ? getObjectReportUrl(object) : "", [object]);
    const handleClick = useCallback((event: any) => {
        // Stop propagation to prevent list item from being selected
        event.stopPropagation();
        // Prevent default to stop href from being used
        event.preventDefault();
        if (link.length === 0) return;
        setLocation(link);
    }, [link, setLocation]);

    return (
        <OuterBox>
            <Link to={link} onClick={handleClick}>
                <ReportIcon fill={palette.background.textSecondary} />
            </Link>
            <CountLabel variant="body1">{reportsCount}</CountLabel>
        </OuterBox>
    );
};
