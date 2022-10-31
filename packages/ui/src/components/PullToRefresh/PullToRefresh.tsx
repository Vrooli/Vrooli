import { Box, IconButton, useTheme } from "@mui/material";
import { RefreshIcon } from "@shared/icons";
import { useEffect, useRef, useState } from "react";

/**
 * Pull-to-refresh component. Needed because iOS PWAs don't support this natively.
 */
export const PullToRefresh = () => {
    const { palette } = useTheme();
    const [iconSize, setIconSize] = useState<number>(20);
    const [willRefresh, setWillRefresh] = useState<boolean>(false);
    const refs = useRef<{ willRefresh: boolean, startX: number | null }>({
        willRefresh: false,
        startX: null,
    });
    // Determine if in standalone (i.e. app is downloaded) mode
    const isStandalone = window.matchMedia('(display-mode: standalone)').matches;

    // Detect scroll
    useEffect(() => {
        const handleScroll = (e: Event) => {
            // Find scroll y position
            const scrollY = window.scrollY;
            // If scrolled far enough upwards, then indicate that the user wants to refresh
            if (scrollY < -175 && !refs.current.willRefresh) {
                setWillRefresh(true);
                refs.current.willRefresh = true;
                setIconSize(32);
            }
            // If scrolling upwards, set icon size from 24 to 32
            else if (scrollY < -50 && !refs.current.willRefresh) {
                setIconSize(Math.round(20 + (scrollY / -14.5)));
            }
            // If scrolled back to 0, then refresh
            if (scrollY === 0 && refs.current.willRefresh) {
                refs.current.willRefresh = false;
                window.location.reload();
            }
        };
        window.addEventListener('scroll', handleScroll);
        return () => {
            window.removeEventListener('scroll', handleScroll);
        };
    }, []);

    if (!isStandalone) return null;
    return (
        <Box
            id='pull-to-refresh'
            sx={{
                position: 'fixed',
                // Must display below the navbar
                top: 0,
                left: 0,
                right: 0,
                paddingTop: { xs: '96px', md: '112px' },
                paddingBottom: 40,
                height: '56px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                zIndex: -1,
                background: palette.background.default,
            }}
        >
            <IconButton sx={{
                background: palette.background.paper,
                // Spin when will refresh
                transform: willRefresh ? 'rotate(360deg)' : 'rotate(0deg)',
                transition: 'transform 500ms cubic-bezier(0.4, 0, 0.2, 1)',
            }}>
                <RefreshIcon
                    fill={palette.background.textPrimary}
                    width={iconSize}
                    height={iconSize}
                />
            </IconButton>
        </Box>
    );
}