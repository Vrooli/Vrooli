import { EllipsisIcon, endpointGetSchedule, Schedule, useLocation } from "@local/shared";
import { Box, IconButton, Tooltip, useTheme } from "@mui/material";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { MouseEvent, useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { OverviewContainer } from "styles";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { ScheduleViewProps } from "../types";

export const ScheduleView = ({
    display = "page",
    onClose,
    partialData,
    zIndex = 200,
}: ScheduleViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { id, isLoading, object: schedule, permissions, setObject: setSchedule } = useObjectFromUrl<Schedule>({
        ...endpointGetSchedule,
        partialData,
    });

    // useEffect(() => {
    //     document.title = `${name} | Vrooli`;
    // }, [name]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const actionData = useObjectActions({
        object: schedule,
        objectType: "Schedule",
        setLocation,
        setObject: setSchedule,
    });

    /**
     * Displays name, avatar, description, and quick links
     */
    const overviewComponent = useMemo(() => (
        <OverviewContainer>
            <Tooltip title="See all options">
                <IconButton
                    aria-label="More"
                    size="small"
                    onClick={openMoreMenu}
                    sx={{
                        display: "block",
                        marginLeft: "auto",
                        marginRight: 1,
                    }}
                >
                    <EllipsisIcon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
        </OverviewContainer>
    ), [palette.background.paper, palette.background.textSecondary, openMoreMenu]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Schedule")}
            />
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={schedule as any}
                onClose={closeMoreMenu}
                zIndex={zIndex + 1}
            />
            <Box sx={{
                background: palette.mode === "light" ? "#b2b3b3" : "#303030",
                display: "flex",
                paddingTop: 5,
                paddingBottom: { xs: 0, sm: 2, md: 5 },
                position: "relative",
            }}>
                {overviewComponent}
            </Box>
            {/* TODO */}
        </>
    );
};
