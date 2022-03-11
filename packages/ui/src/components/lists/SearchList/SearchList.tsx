import { useQuery } from "@apollo/client";
import { Box, Button, CircularProgress, Grid, List, Typography } from "@mui/material";
import { AutocompleteSearchBar, SortMenu, TimeMenu } from "components";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { containerShadow } from "styles";
import {
    AccessTime as TimeIcon,
    Sort as SortListIcon,
} from '@mui/icons-material';
import { SearchQueryVariablesInput, SearchListProps } from "../types";

export function SearchList<DataType, SortBy, Query, QueryVariables extends SearchQueryVariablesInput<SortBy>>({
    searchPlaceholder = 'Search...',
    sortOptions,
    defaultSortOption,
    query,
    take = 20,
    searchString,
    sortBy,
    timeFrame,
    setSearchString,
    setSortBy,
    setTimeFrame,
    listItemFactory,
    getOptionLabel,
    onObjectSelect,
    onScrolledFar,
    where,
    noResultsText = 'No results',
}: SearchListProps<DataType, SortBy>) {
    const [sortAnchorEl, setSortAnchorEl] = useState(null);
    const [timeAnchorEl, setTimeAnchorEl] = useState(null);
    const [sortByLabel, setSortByLabel] = useState<string>(defaultSortOption.label ?? sortOptions.length > 0 ? sortOptions[0].label : 'Sort');
    const [timeFrameLabel, setTimeFrameLabel] = useState<string>('Time');
    const after = useRef<string | undefined>(undefined);
    const createdTimeFrame = useMemo(() => {
        let result: { [x: string]: Date } = {}
        const split = timeFrame?.split(',');
        if (split?.length !== 2) return result;
        const after = new Date(split[0]);
        const before = new Date(split[1]);
        if (Boolean(after.getMonth())) result.after = after;
        if (Boolean(before.getMonth())) result.before = before;
        return result;
    }, [timeFrame]);

    const { data: pageData, refetch: fetchPage, loading } = useQuery<Query, QueryVariables>(query, { variables: ({ input: { after: after.current, take, sortBy, searchString, createdTimeFrame, ...where } } as any) });
    const [allData, setAllData] = useState<DataType[]>([]);

    // On search filters/sort change, reset the page
    useEffect(() => {
        console.log('Resetting page', after.current);
        after.current = undefined;
        fetchPage();
    }, [searchString, sortBy, createdTimeFrame]);

    // Fetch more data by setting "after"
    const loadMore = useCallback(() => {
        console.log('load more!');
        if (!pageData) return;
        const queryData: any = Object.values(pageData)[0];
        if (!queryData || !queryData.pageInfo) return [];
        if (queryData.pageInfo?.hasNextPage) {
            const { endCursor } = queryData.pageInfo;
            if (endCursor) {
                after.current = endCursor;
                fetchPage();
            }
        }
    }, [pageData]);

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
        console.log('parse newly fetched data', pageData);
        const parsedData = parseData(pageData);
        if (!parsedData) {
            setAllData([]);
            return;
        }
        if (after.current) {
            setAllData(curr => [...curr, ...parsedData]);
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
            if (onScrolledFar) onScrolledFar();
        }
    }, [loading, loadMore, onScrolledFar]);

    // Set event listener for infinite scroll
    useEffect(() => {
        window.addEventListener("scroll", handleScroll);
        return () => window.removeEventListener("scroll", handleScroll);
    }, []);

    const handleSearch = useCallback((newString: string) => { setSearchString(newString) }, []);

    const handleSortOpen = (event) => setSortAnchorEl(event.currentTarget);
    const handleSortClose = (label?: string, selected?: string) => {
        console.log('handle sort close', label, selected);
        setSortAnchorEl(null);
        if (selected) setSortBy(selected);
        if (label) setSortByLabel(label);
    };

    const handleTimeOpen = (event) => setTimeAnchorEl(event.currentTarget);
    const handleTimeClose = (label?: string, after?: Date | null, before?: Date | null) => {
        console.log('handle time close', label, after, before);
        setTimeAnchorEl(null);
        if (!after && !before) setTimeFrame(undefined);
        else setTimeFrame(`${after?.getTime()},${before?.getTime()}`);
        if (label) setTimeFrameLabel(label);
    };

    /**
     * When an autocomplete item is selected, navigate to object
     */
    const onInputSelect = useCallback((newValue: any) => {
        console.log('search list oninputselect', newValue)
        if (!newValue) return;
        // Determine object from selected label
        const selectedItem = allData.find(o => (o as any)?.id === newValue?.id);
        if (!selectedItem) return;
        console.log('selectedItem', selectedItem);
        onObjectSelect(selectedItem);
    }, [allData, onObjectSelect]);

    const searchResultContainer = useMemo(() => {
        const hasItems = listItems.length > 0;
        return (
            <Box sx={{
                marginTop: 2,
                ...(hasItems || loading ? {
                    ...containerShadow,
                    background: (t) => t.palette.background.paper,
                    borderRadius: '8px',
                    overflow: 'overlay',
                } : {}),
                ...(loading ? {
                    minHeight: 'min(300px, 25vh)',
                    display: 'flex',
                    justifyContent: 'center',
                    alignItems: 'center',
                } : {
                    display: 'block',
                })
            }}>
                {
                    loading ? (<CircularProgress color="secondary" />) : (
                        hasItems ? (
                            <List sx={{padding: 0}}>
                                {listItems}
                            </List>
                        ) : (<Typography variant="h6" textAlign="center">{noResultsText}</Typography>)
                    )
                }
            </Box>
        )
    }, [listItems, loading]);

    return (
        <>
            <SortMenu
                sortOptions={sortOptions}
                anchorEl={sortAnchorEl}
                onClose={handleSortClose}
            />
            <TimeMenu
                anchorEl={timeAnchorEl}
                onClose={handleTimeClose}
            />
            <Grid container spacing={2} margin="auto" width="min(100%, 600px)">
                <Grid item xs={12} sm={8}>
                    <AutocompleteSearchBar
                        id={`search-bar`}
                        placeholder={searchPlaceholder}
                        options={allData}
                        loading={loading}
                        getOptionKey={getOptionLabel}
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
                        {timeFrameLabel}
                    </Button>
                </Grid>
            </Grid>
            {searchResultContainer}
        </>
    )
}