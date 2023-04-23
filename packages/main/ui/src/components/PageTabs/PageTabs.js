import { jsx as _jsx } from "react/jsx-runtime";
import { Box, Tab, Tabs, Tooltip, useTheme } from "@mui/material";
import { useCallback } from "react";
import { useWindowSize } from "../../utils/hooks/useWindowSize";
export const PageTabs = ({ ariaLabel, currTab, fullWidth = false, onChange, tabs, }) => {
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const handleTabChange = useCallback((event, newValue) => {
        onChange(event, tabs[newValue]);
    }, [onChange, tabs]);
    return (_jsx(Box, { display: "flex", justifyContent: "center", width: "100%", children: _jsx(Tabs, { value: currTab.index, onChange: handleTabChange, indicatorColor: "secondary", textColor: "inherit", variant: (fullWidth && isMobile) ? "fullWidth" : "scrollable", scrollButtons: "auto", allowScrollButtonsMobile: true, "aria-label": ariaLabel, sx: {
                marginBottom: 1,
                paddingLeft: "1em",
                paddingRight: "1em",
                width: (fullWidth && isMobile) ? "100%" : undefined,
            }, children: tabs.map(({ color, href, Icon, label }, index) => {
                const contents = {};
                if (Icon) {
                    contents.icon = _jsx(Icon, { fill: color ?? (isMobile ?
                            palette.primary.contrastText :
                            palette.mode === "dark" ?
                                palette.primary.contrastText :
                                palette.primary.main) });
                }
                else {
                    contents.label = _jsx("span", { style: { color: color ?? "default" }, children: label });
                }
                return (_jsx(Tooltip, { title: Boolean(Icon) ? label : "", children: _jsx(Tab, { id: `${ariaLabel}-${index}`, ...{ "aria-controls": `${ariaLabel}-tabpanel-${index}` }, ...contents, component: href ? "a" : "div", href: href ?? undefined }, index) }));
            }) }) }));
};
//# sourceMappingURL=PageTabs.js.map