import { useQuery } from "@apollo/client";
import { Box, Button, Grid, List, Tooltip, Typography } from "@mui/material";
import { AutocompleteSearchBar, SearchBreadcrumbs, SortMenu, TimeMenu } from "components";
import { CSSProperties, useCallback, useEffect, useMemo, useState } from "react";
import { centeredText, containerShadow, centeredDiv } from "styles";
import { BaseSearchPageProps, SearchQueryVariablesInput } from "./types";
import {
    AccessTime as TimeIcon,
    Sort as SortListIcon,
} from '@mui/icons-material';

export function BaseSearchPage<DataType, SortBy, Query, QueryVariables extends SearchQueryVariablesInput<SortBy>>({
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
    const [searchString, setSearchString] = useState<string>('');
    const [sortAnchorEl, setSortAnchorEl] = useState(null);
    const [timeAnchorEl, setTimeAnchorEl] = useState(null);
    const [sortBy, setSortBy] = useState<SortBy | undefined>(defaultSortOption.value ?? sortOptions.length > 0 ? sortOptions[0].value : undefined);
    const [sortByLabel, setSortByLabel] = useState<string>(defaultSortOption.label ?? sortOptions.length > 0 ? sortOptions[0].label : 'Sort');
    const [createdTimeFrame, setCreatedTimeFrame] = useState<any | undefined>(undefined);
    const [createdTimeFrameLabel, setCreatedTimeFrameLabel] = useState<string>('Time');
    const [after, setAfter] = useState<string | undefined>(undefined);
    const { data: pageData, refetch: fetchPage, loading } = useQuery<Query, QueryVariables>(query, { variables: ({ input: { after, take, sortBy, searchString, createdTimeFrame } } as any) });
    const [allData, setAllData] = useState<DataType[]>([]);
    const [addNewButton, setAddNewButton] = useState<boolean>(false);

    // On search filters/sort change, reset the page
    useEffect(() => {
        setAfter(undefined);
    }, [searchString, sortBy, createdTimeFrame]);

    // Load page whenever "after" is set or unset
    useEffect(() => {
        fetchPage();
    }, [after]);

    // Fetch more data by setting "after"
    const loadMore = useCallback(() => {
        if (!pageData) return;
        const queryData: any = Object.values(pageData)[0];
        if (!queryData || !queryData.pageInfo) return [];
        if (queryData.pageInfo?.hasNextPage) {
            const { endCursor } = queryData.pageInfo;
            if (endCursor) {
                setAfter(endCursor);
            }
        }
    }, [pageData, setAfter]);

    // Helper method for converting fetched data to an array of object data
    const parseData = useCallback((data: any) => {
        if (!data) return [];
        const queryData: any = Object.values(data)[0];
        console.log('query dta', queryData)
        if (!queryData || !queryData.edges) return [];
        return queryData.edges.map((edge, index) => edge.node);
    }, []);

    // Parse newly fetched data, and determine if it should be appended to the existing data
    useEffect(() => {
        const parsedData = parseData(pageData);
        if (!parsedData) {
            setAllData([]);
            return;
        }
        if (after) {
            setAllData([...allData, ...parsedData]);
        } else {
            setAllData(parsedData);
        }
    }, [pageData]);

    const listItems = useMemo(() => allData.map((data, index) => listItemFactory(data, index)), [allData, listItemFactory]);

    // If near the bottom of the page, load more data
    // If scrolled past a certain point, show an "Add New" button
    const handleScroll = useCallback(() => { //TODO THIS DOESN"T WORK YET
        const scrolledY = window.scrollY;
        const windowHeight = window.innerHeight;
        if (!loading && scrolledY > windowHeight - 500) {
            loadMore();
        }
        if (scrolledY > 100) {
            setAddNewButton(true); //TODO have button show up if not enough results for a scroll
        }
    }, [pageData, loadMore, loading]);

    // Set event listener for infinite scroll
    useEffect(() => {
        window.addEventListener("scroll", handleScroll);
        return () => window.removeEventListener("scroll", handleScroll);
    }, []);

    const handleSearch = useCallback((newString) => { console.log('HANDLE SEARCH', newString); setSearchString(newString) }, []);

    const handleSortOpen = (event) => setSortAnchorEl(event.currentTarget);
    const handleSortClose = (label?: string, selected?: string) => {
        setSortAnchorEl(null);
        if (selected) setSortBy(selected as any);
        if (label) setSortByLabel(label);
    };

    const handleTimeOpen = (event) => setTimeAnchorEl(event.currentTarget);
    const handleTimeClose = (label?: string, after?: Date | null, before?: Date | null) => {
        setTimeAnchorEl(null);
        if (!after && !before) setCreatedTimeFrame(undefined);
        else setCreatedTimeFrame({ after, before });
        if (label) setCreatedTimeFrameLabel(label);
    };

    /**
     * When an autocomplete item is selected, navigate to object
     */
    const onInputSelect = useCallback((_e: any, newValue: any) => {
        if (!newValue) return;
        // Determine object from selected label
        const selectedItem = allData.find(o => getOptionLabel(o) === newValue);
        if (!selectedItem) return;
        console.log('selectedItem', selectedItem);
        onObjectSelect(selectedItem);
    }, [allData]);

    const searchResultContainer = useMemo(() => (
        <Box
            sx={{
                ...containerShadow,
                borderRadius: '8px',
                marginTop: 2,
                background: (t) => t.palette.background.default,
            }}
        >
            <List>
                {listItems}
            </List>
        </Box>
    ), [listItems]);

    const addNewButtonContainer = useMemo(() => (
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
                        transform: addNewButton ? 'translateY(0)' : 'translateY(10em)',
                        transition: 'transform 1s ease-in-out',

                    } as CSSProperties}
                >
                    {popupButtonText}
                </Button>
            </Tooltip>
        </Box>
    ), [addNewButton, popupButtonText, popupButtonTooltip, onPopupButtonClick]);

    return (
        <Box id="page">
            <SortMenu
                sortOptions={sortOptions}
                anchorEl={sortAnchorEl}
                onClose={handleSortClose}
            />
            <TimeMenu
                anchorEl={timeAnchorEl}
                onClose={handleTimeClose}
            />
            <SearchBreadcrumbs sx={{ ...centeredDiv, color: (t) => t.palette.secondary.dark }} />
            <Typography component="h2" variant="h4" sx={{ ...centeredText, paddingTop: 2 }}>{title}</Typography>
            <Grid container spacing={2} margin="auto" width="min(100%, 600px)">
                <Grid item xs={12} sm={8}>
                    <AutocompleteSearchBar
                        id={`${title}-search-bar`}
                        placeholder={searchPlaceholder}
                        options={allData}
                        getOptionLabel={getOptionLabel}
                        value={searchString}
                        onChange={handleSearch}
                        onInputChange={onInputSelect}
                    />
                </Grid>
                <Grid item xs={6} sm={2}>
                    <Button
                        color="secondary"
                        fullWidth
                        startIcon={<SortListIcon />}
                        onClick={handleSortOpen}
                        sx={{
                            height: '100%',
                        }}
                    >
                        {sortByLabel}
                    </Button>
                </Grid>
                <Grid item xs={6} sm={2}>
                    <Button
                        color="secondary"
                        fullWidth
                        startIcon={<TimeIcon />}
                        onClick={handleTimeOpen}
                        sx={{
                            height: '100%',
                        }}
                    >
                        {createdTimeFrameLabel}
                    </Button>
                </Grid>
            </Grid>
            {listItems && listItems.length > 0 ? searchResultContainer : null}
            {addNewButtonContainer}
        </Box>
    )
}