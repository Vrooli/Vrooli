import { endpointGetSchedule, Schedule } from "@local/shared";
import { Box, IconButton, Tooltip, useTheme } from "@mui/material";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { EllipsisIcon } from "icons";
import { MouseEvent, useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { OverviewContainer } from "styles";
import { toDisplay } from "utils/display/pageTools";
import { ScheduleViewProps } from "../types";

export const ScheduleView = ({
    isOpen,
    onClose,
}: ScheduleViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const display = toDisplay(isOpen);

    const { id, isLoading, object: schedule, permissions, setObject: setSchedule } = useObjectFromUrl<Schedule>({
        ...endpointGetSchedule,
        objectType: "Schedule",
    });

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
            <Tooltip title={t("MoreOptions")}>
                <IconButton
                    aria-label={t("MoreOptions")}
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
    ), [t, openMoreMenu, palette.background.textSecondary]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Schedule", { count: 1 })}
            />
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={schedule as any}
                onClose={closeMoreMenu}
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
