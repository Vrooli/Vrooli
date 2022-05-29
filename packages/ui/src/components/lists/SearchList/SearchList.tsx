/**
 * Search list for a single object type
 */
import { useLazyQuery } from "@apollo/client";
import { Box, Button, CircularProgress, List, Tooltip, Typography, useTheme } from "@mui/material";
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
import { AutocompleteListItem, getUserLanguages, labelledSortOptions, listToAutocomplete, listToListItems, objectToSearchInfo, parseSearchParams, SortValueToLabelMap, stringifySearchParams, useReactSearch } from "utils";
import { useLocation } from "wouter";

type TimeFrame = {
    after?: Date;
    before?: Date;
}

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
    canSearch = true,
    handleAdd,
    itemKeyPrefix,
    noResultsText = 'No results',
    searchPlaceholder = 'Search...',
    query,
    take = 20,
    objectType,
    onObjectSelect,
    onScrolledFar,
    where,
    session,
}: SearchListProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { advancedSearchSchema, defaultSortBy, sortByOptions } = useMemo(() => objectToSearchInfo[objectType], [objectType]);

    const [sortBy, setSortBy] = useState<string>(defaultSortBy);
    const [searchString, setSearchString] = useState<string>('');
    const [timeFrame, setTimeFrame] = useState<TimeFrame | undefined>(undefined);
    // Handle URL params
    const searchParams = useReactSearch(null);
    useEffect(() => {
        console.log('search params', searchParams);
        if (typeof searchParams.search === 'string') setSearchString(searchParams.search);
        if (typeof searchParams.sort === 'string') setSortBy(searchParams.sort);
        if (typeof searchParams.time === 'object' &&
            !Array.isArray(searchParams.time) &&
            searchParams.time.hasOwnProperty('after') &&
            searchParams.time.hasOwnProperty('before')) {
            console.log('SET FRAME 1. GOT SEARCH PARAMS.TIME BOIIIII', searchParams.time);
            setTimeFrame({
                after: new Date((searchParams.time as any).after),
                before: new Date((searchParams.time as any).before),
            });
        }
    }, [searchParams]);

    const [sortAnchorEl, setSortAnchorEl] = useState(null);
    const [timeAnchorEl, setTimeAnchorEl] = useState(null);
    const [timeFrameLabel, setTimeFrameLabel] = useState<string>('Time');
    const after = useRef<string | undefined>(undefined);

    /**
     * When sort and filter options change, update the URL
     */
    useEffect(() => {
        console.log('basesearch useeffect start', window.location.search, searchString, sortBy, timeFrame)
        const params = parseSearchParams(window.location.search);
        if (searchString) params.search = searchString;
        else delete params.search;
        if (sortBy) params.sort = sortBy;
        else delete params.sort;
        if (timeFrame) {
            params.time = {
                after: timeFrame.after?.toISOString() ?? '',
                before: timeFrame.before?.toISOString() ?? '',
            }
        }
        else delete params.time;
        console.log('basesearch setting location', params, stringifySearchParams(params))
        setLocation(stringifySearchParams(params), { replace: true });
    }, [searchString, sortBy, timeFrame, setLocation]);

    const [advancedSearchParams, setAdvancedSearchParams] = useState<object>({});
    const [getPageData, { data: pageData, loading }] = useLazyQuery<Query, QueryVariables>(query, { 
        variables: ({ 
            input: { 
                after: after.current, 
                take, 
                sortBy, 
                searchString, 
                createdTimeFrame: (timeFrame && Object.keys(timeFrame).length > 0) ? {
                    after: timeFrame.after?.toISOString(),
                    before: timeFrame.before?.toISOString(),
                } : undefined, 
                ...where, 
                ...advancedSearchParams 
            } 
        } as any) });
    const [allData, setAllData] = useState<DataType[]>([]);

    // On search filters/sort change, reset the page
    useEffect(() => {
        console.log('on useeffect search', timeFrame)
        after.current = undefined;
        if (canSearch) getPageData();
    }, [advancedSearchParams, canSearch, searchString, sortBy, timeFrame, where, getPageData]);

    // Fetch more data by setting "after"
    const loadMore = useCallback(() => {
        if (!pageData) return;
        const queryData: any = Object.values(pageData)[0];
        if (!queryData || !queryData.pageInfo) return [];
        if (queryData.pageInfo?.hasNextPage) {
            const { endCursor } = queryData.pageInfo;
            if (endCursor) {
                after.current = endCursor;
                getPageData();
            }
        }
    }, [getPageData, pageData]);

    /**
     * Helper method for converting fetched data to an array of object data
     */
    const parseData = useCallback((data: any) => {
        if (!data) return [];
        const queryData: any = Object.values(data)[0];
        if (!queryData || !queryData.edges) return [];
        return queryData.edges.map((edge, index) => edge.node);
    }, []);

    // Handle advanced search
    useEffect(() => {
        // Open advanced search dialog, if needed
        if (typeof searchParams.advanced === 'boolean') setAdvancedSearchDialogOpen(searchParams.advanced);
        // Any search params that aren't advanced, search, sort, or time MIGHT be advanced search params
        const { advanced, search, sort, time, ...otherParams } = searchParams;
        // Find valid advanced search params
        const allAdvancedSearchParams = advancedSearchSchema.fields.map(f => f.fieldName);
        // fields in both otherParams and allAdvancedSearchParams should be the new advanced search params
        const advancedData = Object.keys(otherParams).filter(k => allAdvancedSearchParams.includes(k));
        setAdvancedSearchParams(advancedData.reduce((acc, k) => ({ ...acc, [k]: otherParams[k] }), {}));
    }, [advancedSearchSchema.fields, searchParams]);

    // Handle advanced search dialog
    const [advancedSearchDialogOpen, setAdvancedSearchDialogOpen] = useState<boolean>(false);
    const handleAdvancedSearchDialogOpen = useCallback(() => { setAdvancedSearchDialogOpen(true) }, []);
    const handleAdvancedSearchDialogClose = useCallback(() => {
        console.log('CLOSE. removingg search params');
        setAdvancedSearchDialogOpen(false)
    }, []);
    const handleAdvancedSearchDialogSubmit = useCallback((values: any) => {
        console.log('SUMIT. setting advanced search params', values);
        // Remove undefined and 0 values
        const valuesWithoutBlanks = Object.fromEntries(Object.entries(values).filter(([_, v]) => v !== undefined && v !== 0));
        // Add advanced search params to url search params
        setLocation(stringifySearchParams({
            ...searchParams,
            ...valuesWithoutBlanks
        }));
        setAdvancedSearchParams(valuesWithoutBlanks);
    }, [searchParams, setLocation]);

    // Parse newly fetched data, and determine if it should be appended to the existing data
    useEffect(() => {
        // Close advanced search dialog
        // handleAdvancedSearchDialogClose();
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
    }), [allData, itemKeyPrefix, loading, session, onObjectSelect])

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
    }, [handleScroll]);

    const handleSearch = useCallback((newString: string) => { setSearchString(newString) }, [setSearchString]);

    const handleSortOpen = (event) => setSortAnchorEl(event.currentTarget);
    const handleSortClose = (label?: string, selected?: string) => {
        setSortAnchorEl(null);
        if (selected) setSortBy(selected);
    };

    const handleTimeOpen = (event) => setTimeAnchorEl(event.currentTarget);
    const handleTimeClose = (label?: string, frame?: { after?: Date | undefined, before?: Date | undefined }) => {
        setTimeAnchorEl(null);
        console.log('SET FRAME 2. HANDLE TIME CLOSEEEEEE', frame)
        setTimeFrame(frame);
        if (label) setTimeFrameLabel(label);
    };

    /**
     * Wrap sortByOptions with labels
     */
    const sortOptionsLabelled = useMemo(() => labelledSortOptions(sortByOptions), [sortByOptions]);

    /**
     * Find sort by label when sortBy changes
     */
    const sortByLabel = useMemo(() => {
        if (sortBy && sortBy in SortValueToLabelMap) return SortValueToLabelMap[sortBy];
        return '';
    }, [sortBy]);

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
    }, [listItems, loading, noResultsText, palette.background.paper]);

    // Update query params
    useEffect(() => {
        let params = parseSearchParams(window.location.search);
        if (advancedSearchDialogOpen) params.advanced = true;
        else delete params.advanced;
        console.log('set heyaaaa', params)
        setLocation(stringifySearchParams(params), { replace: true });
    }, [advancedSearchDialogOpen, setLocation]);

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
                sortOptions={sortOptionsLabelled}
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