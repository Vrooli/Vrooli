import { useQuery } from "@apollo/client";
import { Box, Button, CircularProgress, List, SxProps, Theme, Tooltip, Typography, useTheme } from "@mui/material";
import { AdvancedSearchDialog, AutocompleteSearchBar, SortMenu, TimeMenu } from "components";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { clickSize, containerShadow } from "styles";
import {
    AccessTime as TimeIcon,
    Add as AddIcon,
    Build as AdvancedIcon,
    Sort as SortListIcon,
} from '@mui/icons-material';
import { SearchQueryVariablesInput, SearchListProps } from "../types";
import { AutocompleteListItem, getUserLanguages, listToAutocomplete, listToListItems, parseSearchParams, stringifySearchParams } from "utils";
import { useLocation } from "wouter";

const searchButtonStyle = {
    ...clickSize,
    display: 'flex',
    borderRadius: '50px',
    border: `1px solid ${(t) => t.palette.secondary.main}`,
    margin: 1,
    padding: 1,
    cursor: 'pointer',
    '&:hover': {
        transform: 'scale(1.1)',
    },
    transition: 'transform 0.2s ease-in-out',
};

export function SearchList<DataType, SortBy, Query, QueryVariables extends SearchQueryVariablesInput<SortBy>>({
    defaultSortOption,
    handleAdd,
    itemKeyPrefix,
    noResultsText = 'No results',
    searchPlaceholder = 'Search...',
    setSearchString,
    sortOptions,
    query,
    take = 20,
    searchString,
    sortBy,
    timeFrame,
    setSortBy,
    setTimeFrame,
    onObjectSelect,
    onScrolledFar,
    where,
    session,
}: SearchListProps<SortBy>) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
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

    const [advancedSearchParams, setAdvancedSearchParams] = useState<object>({});
    const { data: pageData, refetch: fetchPage, loading } = useQuery<Query, QueryVariables>(query, { variables: ({ input: { after: after.current, take, sortBy, searchString, createdTimeFrame, ...where, ...advancedSearchParams } } as any) });
    const [allData, setAllData] = useState<DataType[]>([]);

    // On search filters/sort change, reset the page
    useEffect(() => {
        console.log('FETCHIN NEW PAGE DAWG');
        after.current = undefined;
        fetchPage();
    }, [advancedSearchParams, searchString, sortBy, createdTimeFrame, fetchPage, where]);

    // Fetch more data by setting "after"
    const loadMore = useCallback(() => {
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
    }, [pageData, fetchPage]);

    // Helper method for converting fetched data to an array of object data
    const parseData = useCallback((data: any) => {
        if (!data) return [];
        const queryData: any = Object.values(data)[0];
        if (!queryData || !queryData.edges) return [];
        return queryData.edges.map((edge, index) => edge.node);
    }, []);

    // Handle advanced search dialog
    const [advancedSearchDialogOpen, setAdvancedSearchDialogOpen] = useState<boolean>(parseSearchParams(window.location.search).advanced === "true");
    const handleAdvancedSearchDialogOpen = useCallback(() => { setAdvancedSearchDialogOpen(true) }, []);
    const handleAdvancedSearchDialogClose = useCallback(() => {
        console.log('CLOSE. removingg search params');
        setAdvancedSearchDialogOpen(false)
    }, []);
    const handleAdvancedSearchDialogSubmit = useCallback((values: any) => {
        console.log('SUMIT. setting advanced search params', values);
        setAdvancedSearchParams(values);
    }, []);

    // Parse newly fetched data, and determine if it should be appended to the existing data
    useEffect(() => {
        // Close advanced search dialog
        handleAdvancedSearchDialogClose();
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
    }, [pageData, parseData, handleAdvancedSearchDialogClose]);

    const autocompleteOptions: AutocompleteListItem[] = useMemo(() => {
        return listToAutocomplete(allData, getUserLanguages(session)).sort((a: any, b: any) => {
            return b.stars - a.stars;
        });
    }, [allData, session]);

    const listItems = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Routine'), //TODO
        items: allData,
        keyPrefix: itemKeyPrefix,
        loading,
        onClick: (item) => onObjectSelect(item),
        session: session,
    }), [allData, session, itemKeyPrefix, onObjectSelect])

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
        setSortAnchorEl(null);
        if (selected) setSortBy(selected);
        if (label) setSortByLabel(label);
    };

    const handleTimeOpen = (event) => setTimeAnchorEl(event.currentTarget);
    const handleTimeClose = (label?: string, after?: Date | null, before?: Date | null) => {
        setTimeAnchorEl(null);
        if (!after && !before) setTimeFrame(undefined);
        else setTimeFrame(`${after?.getTime()},${before?.getTime()}`);
        if (label) setTimeFrameLabel(label);
    };

    /**
     * When an autocomplete item is selected, navigate to object
     */
    const onInputSelect = useCallback((newValue: any) => {
        if (!newValue) return;
        // Determine object from selected label
        const selectedItem = allData.find(o => (o as any)?.id === newValue?.id);
        if (!selectedItem) return;
        onObjectSelect(selectedItem);
    }, [allData, onObjectSelect]);

    const searchResultContainer = useMemo(() => {
        const hasItems = listItems.length > 0;
        return (
            <Box sx={{
                marginTop: 2,
                maxWidth: '1000px',
                marginLeft: 'auto',
                marginRight: 'auto',
                ...(hasItems ? {
                    ...containerShadow,
                    background: palette.background.paper,
                    borderRadius: '8px',
                    overflow: 'overlay',
                } : {}),
                ...(loading ? {
                    minHeight: '50px',
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
                            <List sx={{ padding: 0 }}>
                                {listItems}
                            </List>
                        ) : (<Typography variant="h6" textAlign="center">{noResultsText}</Typography>)
                    )
                }
            </Box>
        )
    }, [listItems, loading]);

    // Update query params
    useEffect(() => {
        let params = parseSearchParams(window.location.search);
        if (advancedSearchDialogOpen) params.advanced = "true";
        else delete params.advanced;
        setLocation(stringifySearchParams(params), { replace: true });
    }, [advancedSearchDialogOpen]);

    return (
        <>
            {/* Dialog for setting advanced search items */}
            <AdvancedSearchDialog
                handleClose={handleAdvancedSearchDialogClose}
                handleSearch={handleAdvancedSearchDialogSubmit}
                isOpen={advancedSearchDialogOpen}
                session={session}
            />
            {/* Menu for selecting "sort by" type */}
            <SortMenu
                sortOptions={sortOptions}
                anchorEl={sortAnchorEl}
                onClose={handleSortClose}
            />
            {/* Menu for selecting time created */}
            <TimeMenu
                anchorEl={timeAnchorEl}
                onClose={handleTimeClose}
            />
            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                <Box sx={{ width: 'min(100%, 400px)' }}>
                    <AutocompleteSearchBar
                        id={`search-bar`}
                        placeholder={searchPlaceholder}
                        options={autocompleteOptions}
                        loading={loading}
                        getOptionKey={(option: AutocompleteListItem) => option.label ?? ''}
                        getOptionLabel={(option: AutocompleteListItem) => option.label ?? ''}
                        value={searchString}
                        onChange={handleSearch}
                        onInputChange={onInputSelect}
                        session={session}
                    />
                </Box>
            </Box>
            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', marginBottom: 1 }}>
                <Tooltip title="Sort by" placement="top">
                    <Box
                        onClick={handleSortOpen}
                        sx={{ ...searchButtonStyle }}
                    >
                        <SortListIcon sx={{ fill: palette.secondary.main }} />
                        {sortByLabel}
                    </Box>
                </Tooltip>
                <Tooltip title="Time created" placement="top">
                    <Box
                        onClick={handleTimeOpen}
                        sx={{ ...searchButtonStyle }}
                    >
                        <TimeIcon sx={{ fill: palette.secondary.main }} />
                        {timeFrameLabel}
                    </Box>
                </Tooltip>
                <Tooltip title="See all search settings" placement="top">
                    <Box
                        onClick={handleAdvancedSearchDialogOpen}
                        sx={{ ...searchButtonStyle }}
                    >
                        <AdvancedIcon sx={{ fill: palette.secondary.main }} />
                        Advanced
                    </Box>
                </Tooltip>
            </Box>
            {searchResultContainer}
            {/* Add new button */}
            {Boolean(handleAdd) && <Box sx={{
                maxWidth: '400px',
                margin: 'auto',
                paddingTop: 5,
            }}>
                <Button fullWidth onClick={handleAdd} startIcon={<AddIcon />}>Add New</Button>
            </Box>}
        </>
    )
}