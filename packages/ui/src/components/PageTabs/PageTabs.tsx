/* eslint-disable @typescript-eslint/ban-ts-comment */
import { Box, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { PageTabsProps } from "components/types";
import { useWindowSize } from "hooks/useWindowSize";
import { createRef, useCallback, useEffect, useMemo, useRef } from "react";
import { TabStateColors, TabsInfo } from "utils/search/objectToSearch";

const DRAG_END_DELAY_MS = 50;
const RESIZE_UPDATE_UNDERLINE_DELAY_MS = 100;

// Scroll so new tab is centered
function easeInOutCubic(t: number) {
    // eslint-disable-next-line no-magic-numbers
    return t < 0.5 ? 4 * t * t * t : (t - 1) * (2 * t - 2) * (2 * t - 2) + 1;
}

const Underline = styled(Box)(({ theme }) => ({
    position: "absolute",
    bottom: 0,
    left: 0,
    height: "3px",
    backgroundColor: theme.palette.secondary.main,
    transition: "left 0.3s ease, width 0.3s ease",
}));

export function PageTabs<TabList extends TabsInfo>({
    ariaLabel,
    currTab,
    fullWidth = false,
    id,
    ignoreIcons = false,
    onChange,
    tabs,
    sx,
}: PageTabsProps<TabList>) {
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const tabsRef = useRef<HTMLDivElement>(null);
    const tabRefs = useRef(tabs.map(() => createRef<HTMLDivElement>()));
    const underlineRef = useRef<HTMLDivElement>(null);

    // Handle dragging of tabs
    const draggingRef = useRef(false);
    function handleMouseDown(e: React.MouseEvent) {
        const startX = e.clientX;
        const scrollLeft = tabsRef.current?.scrollLeft ?? 0;
        const minimumDistance = 10; // Minimum distance in pixels for a drag
        let distanceMoved = 0; // Track the distance moved


        function handleMouseMove(e: MouseEvent) {
            if (!tabsRef.current) return;

            const x = e.clientX;
            const walk = (x - startX);
            distanceMoved = Math.abs(walk);

            // Only consider it a drag if the mouse has moved more than the minimum distance
            if (distanceMoved > minimumDistance) {
                draggingRef.current = true;
            }
            // But still move the tabs even if it's not considered a drag
            tabsRef.current.scrollLeft = scrollLeft - walk;
        }

        function handleMouseUp() {
            // Only set draggingRef to false if it was considered a drag
            if (distanceMoved > minimumDistance) {
                setTimeout(() => { draggingRef.current = false; }, DRAG_END_DELAY_MS);
            }
            document.removeEventListener("mousemove", handleMouseMove);
            document.removeEventListener("mouseup", handleMouseUp);
        }

        if (tabsRef.current) {
            document.addEventListener("mousemove", handleMouseMove);
            document.addEventListener("mouseup", handleMouseUp);
        }

        e.preventDefault();
    }

    const handleTabChange = useCallback((event: React.SyntheticEvent, newValue: number) => {
        if (draggingRef.current) return;
        // Add transition for smooth animation of underline
        if (underlineRef.current) {
            underlineRef.current.style.transition = "left 0.3s ease, width 0.3s ease";
        }
        // Call onChange prop
        onChange(event, tabs[newValue]);
    }, [onChange, tabs]);

    useEffect(() => {
        if (!tabsRef.current) return;
        const selectedTabRef = tabRefs.current[currTab.index];
        if (!selectedTabRef) return;
        const selectedTab = selectedTabRef.current;
        if (!selectedTab) return;

        const targetScrollPosition = selectedTab.offsetLeft + selectedTab.offsetWidth / 2 - tabsRef.current.offsetWidth / 2;
        const startScrollPosition = tabsRef.current.scrollLeft;
        const duration = 300;
        const startTime = performance.now();

        function animateScroll(time: number) {
            // Calculate the linear progress of the animation
            const linearProgress = Math.min((time - startTime) / duration, 1);

            // Apply the easing function to the linear progress
            const easedProgress = easeInOutCubic(linearProgress);

            const newScrollPosition = startScrollPosition + (targetScrollPosition - startScrollPosition) * easedProgress;
            if (tabsRef.current) tabsRef.current.scrollLeft = newScrollPosition;

            if (linearProgress < 1) {
                requestAnimationFrame(animateScroll);
            }
        }

        requestAnimationFrame(animateScroll);
    }, [currTab.index]);

    const updateUnderline = useCallback(() => {
        if (tabsRef.current && underlineRef.current) {
            const tabsContainer = tabsRef.current;
            const underlineElement = underlineRef.current;
            const activeTab: HTMLElement = tabsContainer.children[currTab.index] as HTMLElement;

            if (activeTab) {
                // Get the margin values
                const margin = getComputedStyle(activeTab).marginLeft || "0";
                const marginValue = parseFloat(margin);

                // Adjust the offsetLeft and offsetWidth by the margin
                const leftPosition = activeTab.offsetLeft - marginValue;
                const width = activeTab.offsetWidth + marginValue * 2;

                underlineElement.style.left = `${leftPosition}px`;
                underlineElement.style.width = `${width}px`;
            }
        }
    }, [currTab.index]);

    useEffect(() => {
        updateUnderline();
    }, [currTab.index, isMobile, updateUnderline]);

    // Update underline on scroll
    useEffect(() => {
        const tabsContainer = tabsRef.current;

        function handleScroll() {
            if (underlineRef.current) {
                underlineRef.current.style.transition = "none";
            }
            updateUnderline();
        }

        if (tabsContainer) {
            tabsContainer.addEventListener("scroll", handleScroll);
            return () => {
                tabsContainer.removeEventListener("scroll", handleScroll);
            };
        }
    }, [updateUnderline]);

    // Update underline on window resize
    useEffect(() => {
        function handleResize() {
            updateUnderline();
        }
        window.addEventListener("resize", handleResize);
        return () => {
            window.removeEventListener("resize", handleResize);
        };
    }, [tabs.length, updateUnderline]);

    // Update underline after short delay when tab length changes
    useEffect(() => {
        const timeout = setTimeout(() => {
            updateUnderline();
        }, RESIZE_UPDATE_UNDERLINE_DELAY_MS);
        return () => {
            clearTimeout(timeout);
        };
    }, [tabs.length, updateUnderline]);

    const outerBoxStyle = useMemo(function outerBoxStyleMemo() {
        return {
            minWidth: fullWidth ? "100%" : undefined,
            maxWidth: "100%",
            display: "flex",
            justifyContent: "flex-start",
            alignItems: "center",
            position: "relative",
            // Hide scrollbars
            overflowX: "scroll",
            whiteSpace: "nowrap",
            scrollbarWidth: "none", // for Firefox
            msOverflowStyle: "none", // for Internet Explorer 11
            "&::-webkit-scrollbar": {
                display: "none", // for Chrome, Safari, and Opera
            },
            // Apply custom styles
            ...sx,
        } as const;
    }, [fullWidth, sx]);

    const tabBoxStyle = useCallback(function tabBoxStyleCallback(isSelected: boolean) {
        return {
            padding: "4px 8px",
            margin: fullWidth ? "0 auto" : "0",
            cursor: "pointer",
            textTransform: "uppercase",
            // Darken non-selected tabs
            filter: isSelected ? "none" : "brightness(0.8)",
        } as const;
    }, [fullWidth]);

    return (
        <Box
            id={id}
            ref={tabsRef}
            role="tablist"
            onMouseDown={handleMouseDown}
            sx={outerBoxStyle}
        >
            {tabs.map(({ color, href, Icon, key, label }, index) => {
                const isSelected = currTab.index === index;
                const providedColor = color !== undefined ?
                    typeof color === "string" ? color :
                        isSelected ?
                            (color as TabStateColors).active :
                            (color as TabStateColors).inactive :
                    undefined;
                const tabColor = providedColor ?? (isMobile ?
                    palette.primary.contrastText :
                    palette.primary.light);
                const labelStyle = { color: tabColor } as const;
                function handleClick(event: React.MouseEvent) {
                    handleTabChange(event, index);
                }
                return (
                    <Tooltip key={key} title={(Icon && !ignoreIcons) ? label : ""}>
                        <Box
                            id={`${ariaLabel}-${index}`}
                            ref={tabRefs.current[index]}
                            role="tab"
                            component={href ? "a" : "div"}
                            href={href}
                            aria-selected={isSelected}
                            aria-controls={`${ariaLabel}-tabpanel-${index}`}
                            onClick={handleClick}
                            style={tabBoxStyle(isSelected)}
                        >
                            {
                                (Icon && !ignoreIcons)
                                    // @ts-ignore TypeScript being bitchy
                                    ? <Icon fill={tabColor} />
                                    : <Typography variant="body2" style={labelStyle}>{label}</Typography>
                            }
                        </Box>
                    </Tooltip>
                );
            })}
            <Underline ref={underlineRef} />
        </Box>
    );
}
