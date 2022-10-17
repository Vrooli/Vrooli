import { Box, IconButton, useTheme } from "@mui/material";
import { RefreshIcon } from "@shared/icons";
import { useEffect, useRef, useState } from "react";

/**
 * Pull-to-refresh component. Needed because iOS PWAs don't support this natively.
 */
export const PullToRefresh = () => {
    const { palette } = useTheme();
    const [willRefresh, setWillRefresh] = useState<boolean>(false);
    const refs = useRef({
        willRefresh: false,
    });
    // Determine if in PWA mode
    const isPWA = window.matchMedia('(display-mode: standalone)').matches;

    // Detect scroll
    useEffect(() => {
        const handleScroll = (e: Event) => {
            console.log('handleScroll', window.scrollY);
            // Find scroll y position
            const scrollY = window.scrollY;
            // If scrolled far enough upwards, then indicate that the user wants to refresh
            if (scrollY < -150) {
                setWillRefresh(true);
                refs.current.willRefresh = true;
            }
            // If scrolled back to 0, then refresh
            if (scrollY === 0 && refs.current.willRefresh) {
                window.location.reload();
            }
        };
        window.addEventListener('scroll', handleScroll);
        return () => {
            window.removeEventListener('scroll', handleScroll);
            setWillRefresh(false);
        };
    }, []);

    if (!isPWA) return null;
    return (
        <Box
            id='pull-to-refresh'
            sx={{
                position: 'fixed',
                // Must display below the navbar
                top: 0,
                left: 0,
                right: 0,
                paddingTop: { xs: '72px', md: '88px' },
                paddingBottom: 40,
                height: '56px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                zIndex: -1,
                background: palette.background.default,
            }}
        >
            <IconButton sx={{ background: palette.background.paper }}>
                <RefreshIcon
                    fill={palette.background.textPrimary}
                    width={willRefresh ? 30 : 24}
                    height={willRefresh ? 30 : 24}
                />
            </IconButton>
        </Box>
    );
}