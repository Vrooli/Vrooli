import { Box, IconButton, Tooltip, useTheme } from "@mui/material";
import { FindByIdInput, Reminder } from "@shared/consts";
import { EllipsisIcon, HelpIcon } from "@shared/icons";
import { useLocation } from '@shared/route';
import { reminderFindOne } from "api/generated/endpoints/reminder_findOne";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { MouseEvent, useCallback, useMemo, useState } from "react";
import { placeholderColor } from "utils/display/listTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { ReminderViewProps } from "../types";

export const ReminderView = ({
    display = 'page',
    partialData,
    zIndex = 200,
}: ReminderViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const profileColors = useMemo(() => placeholderColor(), []);

    const { id, isLoading, object: reminder, permissions, setObject: setReminder } = useObjectFromUrl<Reminder, FindByIdInput>({
        query: reminderFindOne,
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
        object: reminder,
        objectType: 'Reminder',
        setLocation,
        setObject: setReminder,
    });

    /**
     * Displays name, avatar, description, and quick links
     */
    const overviewComponent = useMemo(() => (
        <Box
            position="relative"
            ml='auto'
            mr='auto'
            mt={3}
            bgcolor={palette.background.paper}
            sx={{
                borderRadius: { xs: '0', sm: 2 },
                boxShadow: { xs: 'none', sm: 12 },
                width: { xs: '100%', sm: 'min(500px, 100vw)' }
            }}
        >
            <Box
                width={'min(100px, 25vw)'}
                height={'min(100px, 25vw)'}
                borderRadius='100%'
                position='absolute'
                display='flex'
                justifyContent='center'
                alignItems='center'
                left='50%'
                top="-55px"
                sx={{
                    border: `1px solid black`,
                    backgroundColor: profileColors[0],
                    transform: 'translateX(-50%)',
                }}
            >
                <HelpIcon fill={profileColors[1]} width='80%' height='80%' />
            </Box>
            <Tooltip title="See all options">
                <IconButton
                    aria-label="More"
                    size="small"
                    onClick={openMoreMenu}
                    sx={{
                        display: 'block',
                        marginLeft: 'auto',
                        marginRight: 1,
                    }}
                >
                    <EllipsisIcon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
        </Box >
    ), [palette.background.paper, palette.background.textSecondary, profileColors, openMoreMenu]);

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: 'Reminder',
                }}
            />
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={reminder as any}
                onClose={closeMoreMenu}
                zIndex={zIndex + 1}
            />
            <Box sx={{
                background: palette.mode === 'light' ? "#b2b3b3" : "#303030",
                display: 'flex',
                paddingTop: 5,
                paddingBottom: { xs: 0, sm: 2, md: 5 },
                position: "relative",
            }}>
                {overviewComponent}
            </Box>
            {/* TODO */}
        </>
    )
}