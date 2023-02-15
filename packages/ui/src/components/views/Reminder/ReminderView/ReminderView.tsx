import { Box, IconButton, Tooltip, useTheme } from "@mui/material"
import { useLocation } from '@shared/route';
import { FindByIdOrHandleInput, Reminder } from "@shared/consts";
import { useLazyQuery } from "api/hooks";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import { ObjectActionMenu } from "components";
import { ReminderViewProps } from "../types";
import { parseSingleItemUrl, placeholderColor, useObjectActions } from "utils";
import { EllipsisIcon, HelpIcon } from "@shared/icons";
import { reminderFindOne } from "api/generated/endpoints/reminder";

export const ReminderView = ({
    partialData,
    session,
    zIndex,
}: ReminderViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const profileColors = useMemo(() => placeholderColor(), []);
    // Fetch data
    const urlData = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data, loading }] = useLazyQuery<Reminder, FindByIdOrHandleInput, 'reminder'>(reminderFindOne, 'reminder', { errorPolicy: 'all' });
    const [reminder, setReminder] = useState<Reminder | null | undefined>(null);
    useEffect(() => {
        if (urlData.id || urlData.handle) getData({ variables: urlData })
    }, [getData, urlData]);
    useEffect(() => {
        setReminder(data?.reminder);
    }, [data]);

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
        session,
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
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={reminder as any}
                onClose={closeMoreMenu}
                session={session}
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