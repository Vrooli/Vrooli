import { Box, Tab, Tabs } from "@mui/material";
import { PageTabsProps } from "components/types";
import { useCallback } from "react";

/**
 * Tabs for a page. Ensures that all page tabs are consistent, 
 * and leave room on the sides for drawer swiping
 */
export const PageTabs = <T extends any>({
    ariaLabel,
    currTab,
    onChange,
    tabs,
}: PageTabsProps<T>) => {

    console.log('page tabs', tabs)

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
                variant="scrollable"
                scrollButtons="auto"
                allowScrollButtonsMobile
                aria-label={ariaLabel}
                sx={{
                    marginBottom: 1,
                    paddingLeft: '1em',
                    paddingRight: '1em',
                }}
            >
                {tabs.map((option, index) => (
                    <Tab
                        key={index}
                        id={`${ariaLabel}-${index}`}
                        {...{ 'aria-controls': `${ariaLabel}-tabpanel-${index}` }}
                        label={<span
                            style={{ color: option.color ?? 'default' }}
                        >{option.label}</span>}
                        component={option.href ? 'a' : 'div'}
                        href={option.href ?? undefined}
                    />
                ))}
            </Tabs>
        </Box>
    )
}