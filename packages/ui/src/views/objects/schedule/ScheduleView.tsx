import Box from "@mui/material/Box";
import { useTheme } from "@mui/material/styles";
import { endpointsSchedule, type Schedule } from "@vrooli/shared";
import { useCallback, useMemo, useState, type MouseEvent } from "react";
import { useTranslation } from "react-i18next";
import { IconButton } from "../../../components/buttons/IconButton.js";
import { ObjectActionMenu } from "../../../components/dialogs/ObjectActionMenu/ObjectActionMenu.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { Tooltip } from "../../../components/Tooltip/Tooltip.js";
import { useObjectActions } from "../../../hooks/objectActions.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { IconCommon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { OverviewContainer } from "../../../styles.js";
import { type ScheduleViewProps } from "./types.js";

export function ScheduleView({
    display,
    onClose,
}: ScheduleViewProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { id, isLoading, object: schedule, permissions, setObject: setSchedule } = useManagedObject<Schedule>({
        ...endpointsSchedule.findOne,
        objectType: "Schedule",
    });

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<HTMLElement | null>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<HTMLElement>) => {
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
                    variant="transparent"
                    sx={{
                        display: "block",
                        marginLeft: "auto",
                        marginRight: 1,
                    }}
                >
                    <IconCommon name="Ellipsis" fill="background.textSecondary" />
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
                object={schedule}
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
}
