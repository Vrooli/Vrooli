import { APP_LINKS } from "@local/shared";
import { Box, Button, Tab, Tabs, Tooltip, Typography } from "@mui/material";
import { SearchList } from "components";
import { CSSProperties, useCallback, useEffect, useMemo, useState } from "react";
import { centeredDiv } from "styles";
import { useLocation } from "wouter";
import { BaseSearchPageProps } from "./types";

const tabOptions = [
    ['Organizations', APP_LINKS.SearchOrganizations], 
    ['Projects', APP_LINKS.SearchProjects], 
    ['Routines', APP_LINKS.SearchRoutines], 
    ['Standards', APP_LINKS.SearchStandards], 
    ['Users', APP_LINKS.SearchUsers],
];

export function BaseSearchPage<DataType, SortBy>({
    title = 'Search',
    searchPlaceholder = 'Search...',
    sortOptions,
    defaultSortOption,
    query,
    take = 20,
    listItemFactory,
    getOptionLabel,
    onObjectSelect,
    popupButtonText,
    popupButtonTooltip = "Couldn't find what you were looking for? Try creating your own!",
    onPopupButtonClick,
}: BaseSearchPageProps<DataType, SortBy>) {
    const [, setLocation] = useLocation();
    // Handle tabs
    const [tabIndex, setTabIndex] = useState<number>(() => {
        const tabIndex = tabOptions.findIndex(t => window.location.pathname.startsWith(t[1]));
        return tabIndex >= 0 ? tabIndex : 0;
    });
    const handleTabChange = (event, newValue) => { setTabIndex(newValue) };
    useEffect(() => {
        console.log()
        setLocation(tabOptions[tabIndex][1]);
    }, [tabIndex]);

    // Popup button
    const [popupButton, setPopupButton] = useState<boolean>(false);
    const handleScrolledFar = useCallback(() => { setPopupButton(true) }, [])
    const popupButtonContainer = useMemo(() => (
        <Box sx={{ ...centeredDiv, paddingTop: 1 }}>
            <Tooltip title={popupButtonTooltip}>
                <Button
                    onClick={onPopupButtonClick}
                    size="large"
                    sx={{
                        zIndex: 100,
                        minidth: 'min(100%, 200px)',
                        height: '48px',
                        borderRadius: 3,
                        position: 'fixed',
                        bottom: '5em',
                        transform: popupButton ? 'translateY(0)' : 'translateY(10em)',
                        transition: 'transform 1s ease-in-out',
                    } as CSSProperties}
                >
                    {popupButtonText}
                </Button>
            </Tooltip>
        </Box>
    ), [popupButton, onPopupButtonClick, popupButtonText, popupButtonTooltip]);

    return (
        <Box id="page">
            {/* Navigate between search pages */}
            <Tabs
                value={tabIndex}
                onChange={handleTabChange}
                indicatorColor="secondary"
                textColor="inherit"
                variant="scrollable"
                scrollButtons="auto"
                allowScrollButtonsMobile
                aria-label="search-type-tabs"
                sx={{ 
                    marginBottom: 2,
                    '& .MuiTabs-flexContainer': {
                        justifyContent: 'space-between',
                    },
                }}
            >
                {tabOptions.map((option, index) => (
                    <Tab
                        key={index}
                        id={`search-tab-${index}`}
                        {...{ 'aria-controls': `search-tabpanel-${index}` }}
                        label={option[0]}
                        color={index === 0 ? '#ce6c12' : 'default'}
                    />
                ))}
            </Tabs>
            <Typography component="h2" variant="h4" textAlign="center" sx={{ paddingTop: 2 }}>{title}</Typography>
            <SearchList
                searchPlaceholder={searchPlaceholder}
                sortOptions={sortOptions}
                defaultSortOption={defaultSortOption}
                query={query}
                take={take}
                listItemFactory={listItemFactory}
                getOptionLabel={getOptionLabel}
                onObjectSelect={onObjectSelect}
                onScrolledFar={handleScrolledFar}
            />
            {popupButtonContainer}
        </Box>
    )
}