import { Box, Tooltip, Typography, useTheme } from "@mui/material";
import { PageTabsProps } from "components/types";
import { useCallback, useEffect, useRef } from "react";
import { useWindowSize } from "utils/hooks/useWindowSize";

export const PageTabs = <T extends string | number | object>({
    ariaLabel,
    currTab,
    fullWidth = false,
    id,
    onChange,
    tabs,
    sx,
}: PageTabsProps<T>) => {
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    // Keep selected tab centered
    const tabsRef = useRef<HTMLDivElement>(null);
    const underlineRef = useRef<HTMLDivElement>(null);

    const handleTabChange = useCallback((event: React.SyntheticEvent, newValue: number) => {
        // Add transition for smooth animation of underline
        if (underlineRef.current) {
            underlineRef.current.style.transition = "left 0.3s ease, width 0.3s ease";
        }
        // Call onChange prop
        onChange(event, tabs[newValue]);
    }, [onChange, tabs]);

    // Scroll so new tab is centered
    useEffect(() => {
        const selectedTab = document.getElementById(`${ariaLabel}-${currTab.index}`);
        if (!selectedTab) return;
        selectedTab.scrollIntoView({
            behavior: "smooth",
            block: "center",
            inline: "center",
        });
    }, [ariaLabel, currTab.index]);

    const updateUnderline = useCallback(() => {
        if (tabsRef.current && underlineRef.current) {
            const tabsContainer = tabsRef.current;
            const underlineElement = underlineRef.current;
            const activeTab: HTMLElement = tabsContainer.children[currTab.index] as HTMLElement;

            if (activeTab) {
                // Use the offsetLeft of the active tab
                const leftPosition = activeTab.offsetLeft;
                const width = activeTab.offsetWidth;

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
                    <Tooltip key={index} title={Icon ? label : ""}>
                        <Box
                            id={`${ariaLabel}-${index}`}
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
                            {Icon ? <Icon fill={tabColor} /> : <Typography variant="body2" style={{ color: tabColor }}>{label}</Typography>}
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
