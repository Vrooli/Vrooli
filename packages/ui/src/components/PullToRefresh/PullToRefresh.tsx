import { Box, IconButton, styled, useTheme } from "@mui/material";
import { useEffect, useRef, useState } from "react";
import { IconCommon } from "../../icons/Icons.js";
import { getDeviceInfo } from "../../utils/display/device.js";

const REFRESH_THRESHOLD_PX = 175;
const GROW_THRESHOLD_PX = 50;
const GROW_RATE = 14.5;
const START_SIZE_PX = 20;
const END_SIZE_PX = 32;

const OuterBox = styled(Box)(({ theme }) => ({
    position: "fixed",
    // Must display below the navbar
    top: 0,
    left: 0,
    right: 0,
    paddingTop: "112px",
    paddingBottom: 40,
    height: "56px",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    zIndex: -1, // Below the main content, but above the page and footer's hidden divs (for coloring overscroll)
    background: theme.palette.background.default,
    [theme.breakpoints.down("md")]: {
        paddingTop: "96px",
    },
}));

/**
 * Pull-to-refresh component. Needed because iOS PWAs don't support this natively.
 */
export function PullToRefresh() {
    const { palette } = useTheme();
    const [iconSize, setIconSize] = useState<number>(START_SIZE_PX);
    const [willRefresh, setWillRefresh] = useState<boolean>(false);
    const refs = useRef<{ willRefresh: boolean, startX: number | null }>({
        willRefresh: false,
        startX: null,
    });
    // Determine if in standalone (i.e. app is downloaded)
    const { isStandalone } = getDeviceInfo();

    useEffect(function detectScrollEffect() {
        function handleScroll() {
            // Find scroll y position
            const scrollY = window.scrollY;
            // If scrolled far enough upwards, then indicate that the user wants to refresh
            if (scrollY < -REFRESH_THRESHOLD_PX && !refs.current.willRefresh) {
                setWillRefresh(true);
                refs.current.willRefresh = true;
                setIconSize(END_SIZE_PX);
            }
            // If scrolling upwards, set icon size from 24 to 32
            else if (scrollY < -GROW_THRESHOLD_PX && !refs.current.willRefresh) {
                setIconSize(Math.round(START_SIZE_PX + (scrollY / -GROW_RATE)));
            }
            // If scrolled back to 0, then refresh
            if (scrollY === 0 && refs.current.willRefresh) {
                refs.current.willRefresh = false;
                window.location.reload();
            }
        }
        window.addEventListener("scroll", handleScroll);
        return () => {
            window.removeEventListener("scroll", handleScroll);
        };
    }, []);

    if (!isStandalone) return null;
    return (
        <OuterBox id="pull-to-refresh">
            <IconButton sx={{
                background: palette.background.paper,
                // Spin when will refresh
                transform: willRefresh ? "rotate(360deg)" : "rotate(0deg)",
                transition: "transform 500ms cubic-bezier(0.4, 0, 0.2, 1)",
            }}>
                <IconCommon
                    decorative
                    fill={palette.background.textPrimary}
                    name="Refresh"
                    size={iconSize}
                />
            </IconButton>
        </OuterBox>
    );
}
