import { Box, Tab, Tabs, Tooltip, useTheme } from "@mui/material";
import { PageTabsProps } from "components/types";
import { useCallback } from "react";
import { useWindowSize } from "utils";

/**
 * Tabs for a page. Ensures that all page tabs are consistent, 
 * and leave room on the sides for drawer swiping
 */
export const PageTabs = <T extends any>({
    ariaLabel,
    currTab,
    fullWidth = false,
    onChange,
    tabs,
}: PageTabsProps<T>) => {
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const handleTabChange = useCallback((event: React.SyntheticEvent, newValue: number) => {
        onChange(event, tabs[newValue]);
    }, [onChange, tabs]);

    return (
        <Box display="flex" justifyContent="center" width="100%">
            <Tabs
                value={currTab.index}
                onChange={handleTabChange}
                indicatorColor="secondary"
                textColor="inherit"
                variant={(fullWidth && isMobile) ? "fullWidth" : "scrollable"}
                scrollButtons="auto"
                allowScrollButtonsMobile
                aria-label={ariaLabel}
                sx={{
                    marginBottom: 1,
                    paddingLeft: '1em',
                    paddingRight: '1em',
                    width: (fullWidth && isMobile) ? '100%' : undefined,
                }}
            >
                {tabs.map(({ color, href, Icon, label }, index) => {
                    // If icon is provided, use it. Otherwise, use the label
                    const contents: { [x: string]: any } = {};
                    if (Icon) {
                        contents.icon = <Icon fill={isMobile ? palette.primary.contrastText : palette.mode === 'dark' ? palette.primary.contrastText : palette.primary.main} />;
                    } else {
                        contents.label = <span style={{ color: color ?? 'default' }}>{label}</span>;
                    }
                    return (<Tooltip title={Boolean(Icon) ? label : ''}>
                        <Tab
                            key={index}
                            id={`${ariaLabel}-${index}`}
                            {...{ 'aria-controls': `${ariaLabel}-tabpanel-${index}` }}
                            {...contents}
                            component={href ? 'a' : 'div'}
                            href={href ?? undefined}
                        />
                    </Tooltip>)
                })}
            </Tabs>
        </Box>
    )
}