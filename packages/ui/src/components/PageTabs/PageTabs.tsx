import { Box, Tooltip, Typography, useTheme } from "@mui/material";
import { PageTabsProps } from "components/types";
import { createRef, useCallback, useEffect, useRef } from "react";
import { useWindowSize } from "utils/hooks/useWindowSize";

export const PageTabs = <T extends string | number | object>({
    ariaLabel,
    currTab,
    fullWidth = false,
    id,
    ignoreIcons = false,
    onChange,
    tabs,
    sx,
}: PageTabsProps<T>) => {
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const tabsRef = useRef<HTMLDivElement>(null);
    const tabRefs = useRef(tabs.map(() => createRef<HTMLDivElement>()));
    const underlineRef = useRef<HTMLDivElement>(null);

    // Handle dragging of tabs
    const draggingRef = useRef(false);
    const handleMouseDown = (e: React.MouseEvent) => {
        if (tabsRef.current) {
            const startX = e.clientX;
            const scrollLeft = tabsRef.current.scrollLeft;
            const minimumDistance = 10; // Minimum distance in pixels for a drag
            let distanceMoved = 0; // Track the distance moved

            const handleMouseMove = (e: MouseEvent) => {
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
            };

            const handleMouseUp = () => {
                // Only set draggingRef to false if it was considered a drag
                if (distanceMoved > minimumDistance) {
                    setTimeout(() => { draggingRef.current = false; }, 50);
                }
                document.removeEventListener("mousemove", handleMouseMove);
                document.removeEventListener("mouseup", handleMouseUp);
            };

            document.addEventListener("mousemove", handleMouseMove);
            document.addEventListener("mouseup", handleMouseUp);
        }

        e.preventDefault();
    };

    const handleTabChange = useCallback((event: React.SyntheticEvent, newValue: number) => {
        if (draggingRef.current) return;
        // Add transition for smooth animation of underline
        if (underlineRef.current) {
            underlineRef.current.style.transition = "left 0.3s ease, width 0.3s ease";
        }
        // Call onChange prop
        onChange(event, tabs[newValue]);
    }, [onChange, tabs]);

    // Scroll so new tab is centered
    const easeInOutCubic = (t: number) => t < 0.5 ? 4 * t * t * t : (t - 1) * (2 * t - 2) * (2 * t - 2) + 1;
    useEffect(() => {
        const selectedTab = tabRefs.current[currTab.index].current;
        if (!selectedTab || !tabsRef.current) return;

        const targetScrollPosition = selectedTab.offsetLeft + selectedTab.offsetWidth / 2 - tabsRef.current.offsetWidth / 2;
        const startScrollPosition = tabsRef.current.scrollLeft;
        const duration = 300;
        const startTime = performance.now();

        const animateScroll = (time: number) => {
            // Calculate the linear progress of the animation
            const linearProgress = Math.min((time - startTime) / duration, 1);

            // Apply the easing function to the linear progress
            const easedProgress = easeInOutCubic(linearProgress);

            const newScrollPosition = startScrollPosition + (targetScrollPosition - startScrollPosition) * easedProgress;
            if (tabsRef.current) tabsRef.current.scrollLeft = newScrollPosition;

            if (linearProgress < 1) {
                requestAnimationFrame(animateScroll);
            }
        };

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
        if (tabsContainer) {
            const handleScroll = () => {
                if (underlineRef.current) {
                    underlineRef.current.style.transition = "none";
                }
                updateUnderline();
            };
            tabsContainer.addEventListener("scroll", handleScroll);
            return () => {
                tabsContainer.removeEventListener("scroll", handleScroll);
            };
        }
    }, [updateUnderline]);

    // Update underline on window resize
    useEffect(() => {
        const handleResize = () => {
            updateUnderline();
        };
        window.addEventListener("resize", handleResize);
        return () => {
            window.removeEventListener("resize", handleResize);
        };
    }, [tabs.length, updateUnderline]);

    // Update underline after short delay when tab length changes
    useEffect(() => {
        const timeout = setTimeout(() => {
            updateUnderline();
        }, 100);
        return () => {
            clearTimeout(timeout);
        };
    }, [tabs.length, updateUnderline]);

    return (
        <Box
            id={id}
            ref={tabsRef}
            role="tablist"
            onMouseDown={handleMouseDown}
            sx={{
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
            }}
        >
            {tabs.map(({ color, href, Icon, label }, index) => {
                const isSelected = currTab.index === index;
                const tabColor = color ?? (isMobile ?
                    palette.primary.contrastText :
                    palette.primary.light);
                return (
                    <Tooltip key={index} title={(Icon && !ignoreIcons) ? label : ""}>
                        <Box
                            id={`${ariaLabel}-${index}`}
                            ref={tabRefs.current[index]}
                            role="tab"
                            component={href ? "a" : "div"}
                            href={href}
                            aria-selected={isSelected}
                            aria-controls={`${ariaLabel}-tabpanel-${index}`}
                            onClick={(event) => handleTabChange(event, index)}
                            style={{
                                padding: "10px",
                                margin: fullWidth ? "0 auto" : "0",
                                cursor: "pointer",
                                textTransform: "uppercase",
                                // Darken non-selected tabs
                                filter: isSelected ? "none" : "brightness(0.8)",
                            }}
                        >
                            {(Icon && !ignoreIcons) ? <Icon fill={tabColor} /> : <Typography variant="body2" style={{ color: tabColor }}>{label}</Typography>}
                        </Box>
                    </Tooltip>
                );
            })}
            <div
                ref={underlineRef}
                style={{
                    position: "absolute",
                    bottom: 0,
                    left: 0,
                    height: "3px",
                    backgroundColor: palette.secondary.main,
                    transition: "left 0.3s ease, width 0.3s ease",
                }}
            />
        </Box>
    );
};
