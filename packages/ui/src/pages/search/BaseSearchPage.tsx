import { APP_LINKS } from "@local/shared";
import { Box, Button, IconButton, Stack, Tab, Tabs, Tooltip, Typography } from "@mui/material";
import { SearchList } from "components";
import { useCallback, useEffect, useMemo, useState } from "react";
import { centeredDiv } from "styles";
import { useLocation } from "wouter";
import { BaseSearchPageProps } from "./types";
import { Add as AddIcon } from '@mui/icons-material';
import { parseSearchParams } from "utils/urlTools";

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
    showAddButton = true,
    onAddClick = () => {},
    popupButtonText,
    popupButtonTooltip = "Couldn't find what you were looking for? Try creating your own!",
    onPopupButtonClick,
}: BaseSearchPageProps<DataType, SortBy>) {
    const [, setLocation] = useLocation();
    // Handle url search
    const [searchString, setSearchString] = useState<string>(parseSearchParams(window.location.search).search ?? '');
    const [sortBy, setSortBy] = useState<string | undefined>(parseSearchParams(window.location.search).sort ?? defaultSortOption.value ?? sortOptions.length > 0 ? sortOptions[0].value : undefined);
    const [timeFrame, setTimeFrame] = useState<string | undefined>(parseSearchParams(window.location.search).time);
    useEffect(() => {
        let searchUrl = '?';
        if (searchString) searchUrl += `search=${searchString}&`;
        if (sortBy) searchUrl += `sort=${sortBy}&`;
        if (timeFrame) searchUrl += `time=${timeFrame}&`;
        setLocation(searchUrl.substring(0, searchUrl.length - 1), { replace: true });
    }, [searchString, sortBy, timeFrame, setLocation]);
    // Handle tabs
    const tabIndex = useMemo(() => {
        const index = tabOptions.findIndex(t => window.location.pathname.startsWith(t[1]));
        return Math.max(index, 0);
    }, [window.location.pathname]);
    const handleTabChange = (_e, newIndex) => { 
        setLocation(tabOptions[newIndex][1], { replace: true });
    };

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
                        minWidth: 'min(100%, 200px)',
                        height: '48px',
                        borderRadius: 3,
                        position: 'fixed',
                        bottom: '5em',
                        transform: popupButton ? 'translateY(0)' : 'translateY(10em)',
                        transition: 'transform 1s ease-in-out',
                    }}
                >
                    {popupButtonText}
                </Button>
            </Tooltip>
        </Box>
    ), [popupButton, popupButtonText, popupButtonTooltip, onPopupButtonClick]);

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
            <Stack direction="row" alignItems="center" justifyContent="center" sx={{ paddingTop: 2 }}>
                <Typography component="h2" variant="h4">{title}</Typography>
                { showAddButton ? <Tooltip title="Add new" placement="top">
                    <IconButton size="large" onClick={onAddClick} sx={{padding: 1}}>
                        <AddIcon color="primary" sx={{width: '1.5em', height: '1.5em'}} />
                    </IconButton>
                </Tooltip> : null }
            </Stack>
            <SearchList
                searchPlaceholder={searchPlaceholder}
                sortOptions={sortOptions}
                defaultSortOption={defaultSortOption}
                query={query}
                take={take}
                searchString={searchString}
                sortBy={sortBy}
                timeFrame={timeFrame}
                setSearchString={setSearchString}
                setSortBy={setSortBy}
                setTimeFrame={setTimeFrame}
                listItemFactory={listItemFactory}
                getOptionLabel={getOptionLabel}
                onObjectSelect={onObjectSelect}
                onScrolledFar={handleScrolledFar}
            />
            {popupButtonContainer}
        </Box >
    )
}