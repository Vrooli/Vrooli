/**
 * Search list for a single object type
 */
import { useLazyQuery } from "@apollo/client";
import { Box, Button, List, Tooltip, Typography, useTheme } from "@mui/material";
import { AdvancedSearchDialog, AutocompleteSearchBar, SortMenu, TimeMenu } from "components";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { clickSize, containerShadow } from "styles";
import {
    Sort as SortListIcon,
} from '@mui/icons-material';
import { BuildIcon, HistoryIcon as TimeIcon, PlusIcon } from '@shared/icons';
import { SearchQueryVariablesInput, SearchListProps } from "../types";
import { addSearchParams, getUserLanguages, labelledSortOptions, listToAutocomplete, listToListItems, parseSearchParams, removeSearchParams, SearchParams, searchTypeToParams, SortValueToLabelMap } from "utils";
import { useLocation } from '@shared/route';
import { AutocompleteOption } from "types";

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

/**
 * Helper method for converting fetched data to an array of object data
 */
const parseData = (data: any) => {
    if (!data) return [];
    const queryData: any = Object.values(data)[0];
    if (!queryData || !queryData.edges) return [];
    return queryData.edges.map((edge, index) => edge.node);
};

export function SearchList<DataType, SortBy, Query, QueryVariables extends SearchQueryVariablesInput<SortBy>>({
    canSearch = true,
    handleAdd,
    hideRoles,
    id,
    itemKeyPrefix,
    noResultsText = 'No results',
    searchPlaceholder = 'Search...',
    take = 20,
    searchType,
    onObjectSelect,
    onScrolledFar,
    where,
    session,
    zIndex,
}: SearchListProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { advancedSearchSchema, defaultSortBy, sortByOptions, query } = useMemo<SearchParams>(() => searchTypeToParams[searchType], [searchType]);

    const [sortBy, setSortBy] = useState<string>(defaultSortBy);
    const [searchString, setSearchString] = useState<string>('');
    const [timeFrame, setTimeFrame] = useState<TimeFrame | undefined>(undefined);
    useEffect(() => {
        const searchParams = parseSearchParams(window.location.search)
        if (typeof searchParams.search === 'string') setSearchString(searchParams.search);
        if (typeof searchParams.sort === 'string') {
            // Check if sortBy is valid
            if (searchParams.sort in sortByOptions) {
                setSortBy(searchParams.sort);
            } else {
                setSortBy(defaultSortBy);
            }
        }
        if (typeof searchParams.time === 'object' &&
            !Array.isArray(searchParams.time) &&
            searchParams.time.hasOwnProperty('after') &&
            searchParams.time.hasOwnProperty('before')) {
            setTimeFrame({
                after: new Date((searchParams.time as any).after),
                before: new Date((searchParams.time as any).before),
            });
        }
    }, [defaultSortBy, searchType, sortByOptions]);

    const [sortAnchorEl, setSortAnchorEl] = useState<HTMLElement | null>(null);
    const [timeAnchorEl, setTimeAnchorEl] = useState<HTMLElement | null>(null);
    const [timeFrameLabel, setTimeFrameLabel] = useState<string>('Time');
    const after = useRef<string | undefined>(undefined);

    /**
     * When sort and filter options change, update the URL
     */
    useEffect(() => {
        addSearchParams(setLocation, {
            search: searchString.length > 0 ? searchString : undefined,
            sort: sortBy,
            time: timeFrame ? {
                after: timeFrame.after?.toISOString() ?? '',
                before: timeFrame.before?.toISOString() ?? '',
            } : undefined,
        });
    }, [searchString, sortBy, timeFrame, setLocation]);

    const [advancedSearchParams, setAdvancedSearchParams] = useState<object | null>(null);
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
        } as any),
        errorPolicy: 'all',
    });
    const [allData, setAllData] = useState<DataType[]>([]);

    // On search filters/sort change, reset the page
    useEffect(() => {
        after.current = undefined;
        if (canSearch) getPageData();
    }, [advancedSearchParams, canSearch, searchString, searchType, sortBy, timeFrame, where, getPageData]);

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

    // Handle advanced search
    useEffect(() => {
        const searchParams = parseSearchParams(window.location.search)
        if (!advancedSearchSchema?.fields) {
            setAdvancedSearchParams(null);
            return;
        }
        // Open advanced search dialog, if needed
        if (typeof searchParams.advanced === 'boolean') setAdvancedSearchDialogOpen(searchParams.advanced);
        // Any search params that aren't advanced, search, sort, or time MIGHT be advanced search params
        const { advanced, search, sort, time, ...otherParams } = searchParams;
        // Find valid advanced search params
        const allAdvancedSearchParams = advancedSearchSchema.fields.map(f => f.fieldName);
        // fields in both otherParams and allAdvancedSearchParams should be the new advanced search params
        const advancedData = Object.keys(otherParams).filter(k => allAdvancedSearchParams.includes(k));
        setAdvancedSearchParams(advancedData.reduce((acc, k) => ({ ...acc, [k]: otherParams[k] }), {}));
    }, [advancedSearchSchema?.fields]);

    // Handle advanced search dialog
    const [advancedSearchDialogOpen, setAdvancedSearchDialogOpen] = useState<boolean>(false);
    const handleAdvancedSearchDialogOpen = useCallback(() => { setAdvancedSearchDialogOpen(true) }, []);
    const handleAdvancedSearchDialogClose = useCallback(() => {
        setAdvancedSearchDialogOpen(false)
    }, []);
    const handleAdvancedSearchDialogSubmit = useCallback((values: any) => {
        // Remove 0 values
        const valuesWithoutBlanks: { [x: string]: any } = Object.fromEntries(Object.entries(values).filter(([_, v]) => v !== 0));
        // Remove schema fields from search params
        removeSearchParams(setLocation, advancedSearchSchema?.fields?.map(f => f.fieldName) ?? []);
        // Add set fields to search params
        addSearchParams(setLocation, valuesWithoutBlanks);
        setAdvancedSearchParams(valuesWithoutBlanks);
    }, [advancedSearchSchema?.fields, setLocation]);

    // Parse newly fetched data, and determine if it should be appended to the existing data
    useEffect(() => {
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
    }, [pageData, handleAdvancedSearchDialogClose]);

    const autocompleteOptions: AutocompleteOption[] = useMemo(() => {
        return listToAutocomplete(allData as any, getUserLanguages(session)).sort((a: any, b: any) => {
            return b.stars - a.stars;
        });
    }, [allData, session]);

    const listItems = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill(searchType),
        hideRoles,
        items: (allData.length > 0 ? allData : parseData(pageData)) as any[],
        keyPrefix: itemKeyPrefix,
        loading,
        onClick: (item) => onObjectSelect(item),
        session: session,
    }), [searchType, hideRoles, allData, pageData, itemKeyPrefix, loading, session, onObjectSelect])
    console.log('listItems', new Date().getTime(), listItems, loading, pageData)

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
        console.log('searchResultsContainer', new Date().getTime(), hasItems, listItems);
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
                    display: 'block',
                } : {}),
            }}>
                {
                    hasItems ? (
                        <List sx={{ padding: 0 }}>
                            {listItems}
                        </List>
                    ) : (<Typography variant="h6" textAlign="center">{noResultsText}</Typography>)
                }
            </Box>
        )
    }, [listItems, noResultsText, palette.background.paper]);

    // Update query params
    useEffect(() => {
        addSearchParams(setLocation, { advanced: advancedSearchDialogOpen });
    }, [advancedSearchDialogOpen, setLocation]);

    return (
        <>
            {/* Dialog for setting advanced search items */}
            <AdvancedSearchDialog
                handleClose={handleAdvancedSearchDialogClose}
                handleSearch={handleAdvancedSearchDialogSubmit}
                isOpen={advancedSearchDialogOpen}
                searchType={searchType}
                session={session}
                zIndex={zIndex + 1}
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
                        id={`search-bar-${id}`}
                        placeholder={searchPlaceholder}
                        options={autocompleteOptions}
                        loading={loading}
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
                        <TimeIcon fill={palette.secondary.main} />
                        {timeFrameLabel}
                    </Box>
                </Tooltip>
                {advancedSearchParams && <Tooltip title="See all search settings" placement="top">
                    <Box
                        onClick={handleAdvancedSearchDialogOpen}
                        sx={{ ...searchButtonStyle }}
                    >
                        <BuildIcon fill={palette.secondary.main} />
                        Advanced
                    </Box>
                </Tooltip>}
            </Box>
            {searchResultContainer}
            {/* Add new button */}
            {Boolean(handleAdd) && <Box sx={{
                maxWidth: '400px',
                margin: 'auto',
                paddingTop: 5,
            }}>
                <Button fullWidth onClick={handleAdd} startIcon={<PlusIcon />}>Add New</Button>
            </Box>}
        </>
    )
}