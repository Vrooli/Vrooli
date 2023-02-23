/**
 * Search list for a single object type
 */
import { useLazyQuery } from "api/hooks";
import { Box, Button, List, Palette, Tooltip, Typography, useTheme } from "@mui/material";
import { AdvancedSearchDialog, SiteSearchBar, SortMenu, TimeMenu } from "components";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { BuildIcon, HistoryIcon as TimeIcon, PlusIcon, SortIcon } from '@shared/icons';
import { SearchQueryVariablesInput, SearchListProps } from "../types";
import { getUserLanguages, labelledSortOptions, ListObjectType, listToAutocomplete, listToListItems, openObject, searchTypeToParams, useDisplayApolloError } from "utils";
import { addSearchParams, parseSearchParams, removeSearchParams, useLocation } from '@shared/route';
import { AutocompleteOption } from "types";
import { SearchParams } from "utils/search/schemas/base";
import { routineFindMany } from "api/generated/endpoints/routine";
import { useTranslation } from "react-i18next";
import { exists } from "@shared/utils";

type TimeFrame = {
    after?: Date;
    before?: Date;
}

const searchButtonStyle = (palette: Palette) => ({
    minHeight: '34px',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    borderRadius: '50px',
    border: `2px solid ${palette.secondary.main}`,
    margin: 1,
    padding: 0,
    paddingLeft: 1,
    paddingRight: 1,
    cursor: 'pointer',
    '&:hover': {
        transform: 'scale(1.1)',
    },
    transition: 'transform 0.2s ease-in-out',
});

/**
 * Helper method for converting fetched data to an array of object data
 */
const parseData = (data: any) => {
    if (!data) return [];
    const queryData: any = Object.values(data)[0];
    if (!queryData || !queryData.edges) return [];
    return queryData.edges.map((edge, index) => edge.node);
};

export function SearchList<
    DataType extends ListObjectType,
    SortBy,
    Endpoint extends string,
    QueryResult extends string | number | boolean | object,
    QueryVariables extends SearchQueryVariablesInput<SortBy>
>({
    beforeNavigation,
    canSearch = true,
    handleAdd,
    hideRoles,
    id,
    searchPlaceholder,
    take = 20,
    searchType,
    onScrolledFar,
    where,
    session,
    zIndex,
}: SearchListProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const [{ advancedSearchSchema, defaultSortBy, endpoint, sortByOptions, query }, setSearchParams] = useState<Partial<SearchParams>>({});
    useEffect(() => {
        const fetchParams = async () => {
            const params = searchTypeToParams[searchType];
            if (!params) return;
            setSearchParams(await params(getUserLanguages(session)[0]));
        };
        fetchParams();
    }, [searchType, session]);

    const [sortBy, setSortBy] = useState<string>(defaultSortBy);
    const [searchString, setSearchString] = useState<string>('');
    const [timeFrame, setTimeFrame] = useState<TimeFrame | undefined>(undefined);
    useEffect(() => {
        const searchParams = parseSearchParams()
        if (typeof searchParams.search === 'string') setSearchString(searchParams.search);
        if (typeof searchParams.sort === 'string') {
            // Check if sortBy is valid
            if (exists(sortByOptions) && searchParams.sort in sortByOptions) {
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
    const [timeFrameLabel, setTimeFrameLabel] = useState<string>('');
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
    const [getPageData, { data: pageData, loading, error }] = useLazyQuery<QueryResult, QueryVariables, Endpoint>(query ?? routineFindMany, (endpoint ?? 'routines') as any, { // We have to set something as the defaults, so I picked routines
        variables: ({
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
        } as any),
        errorPolicy: 'all',
    });
    useDisplayApolloError(error);
    const [allData, setAllData] = useState<DataType[]>(() => {
        // Check if we just navigated back to this page from an object page
        const lastPath = sessionStorage.getItem("lastPath");
        const lastSearchParams = sessionStorage.getItem("lastSearchParams");
        console.log('lastPath', lastPath)
        console.log('lastSearchParams', lastSearchParams)
        return []
    });

    console.log('allData', allData.length, allData)

    // On search filters/sort change, reset the page
    useEffect(() => {
        after.current = undefined;
        if (canSearch) getPageData();
    }, [advancedSearchParams, canSearch, searchString, searchType, sortBy, timeFrame, where, getPageData]);

    // Fetch more data by setting "after"
    const loadMore = useCallback(() => {
        if (!pageData) return;
        console.log('in load more')
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
        const searchParams = parseSearchParams()
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
        console.log('parsing data', parsedData, after.current)
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
            return b.bookmarks - a.bookmarks;
        });
    }, [allData, session]);

    const listItems = useMemo(() => listToListItems({
        beforeNavigation,
        dummyItems: new Array(5).fill(searchType),
        hideRoles,
        items: (allData.length > 0 ? allData : parseData(pageData)) as any[],
        keyPrefix: `${searchType}-list-item`,
        loading,
        session: session,
        zIndex,
    }), [beforeNavigation, searchType, hideRoles, allData, pageData, loading, session, zIndex])

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

    const handleSearch = useCallback((newString: any) => { setSearchString(newString) }, [setSearchString]);

    const handleSortOpen = (event) => setSortAnchorEl(event.currentTarget);
    const handleSortClose = (label?: string, selected?: string) => {
        setSortAnchorEl(null);
        if (selected) setSortBy(selected);
    };

    const handleTimeOpen = (event) => setTimeAnchorEl(event.currentTarget);
    const handleTimeClose = (label?: string, frame?: { after?: Date | undefined, before?: Date | undefined }) => {
        setTimeAnchorEl(null);
        setTimeFrame(frame);
        if (label) setTimeFrameLabel(label === 'All Time' ? '' : label);
    };

    /**
     * Wrap sortByOptions with labels
     */
    const sortOptionsLabelled = useMemo(() => labelledSortOptions(sortByOptions, getUserLanguages(session)[0]), [session, sortByOptions]);

    /**
     * Find sort by label when sortBy changes
     */
    const sortByLabel = useMemo(() => t(`common:${sortBy}`, { lng: getUserLanguages(session)[0] }) ?? sortBy, [session, sortBy, t]);

    /**
     * When an autocomplete item is selected, navigate to object
     */
    const onInputSelect = useCallback((newValue: any) => {
        if (!newValue) return;
        // Determine object from selected label
        const selectedItem = allData.find(o => (o as any)?.id === newValue?.id);
        if (!selectedItem) return;
        openObject(selectedItem, setLocation);
    }, [allData, setLocation]);

    const searchResultContainer = useMemo(() => {
        const hasItems = listItems.length > 0;
        return (
            <Box sx={{
                marginTop: 2,
                maxWidth: '1000px',
                marginLeft: 'auto',
                marginRight: 'auto',
                ...(hasItems ? {
                    boxShadow: 12,
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
                    ) : (<Typography variant="h6" textAlign="center">{t(`error:NoResults`, { lng: getUserLanguages(session)[0] })}</Typography>)
                }
            </Box>
        )
    }, [listItems, palette.background.paper, session, t]);

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
                lng={getUserLanguages(session)[0]}
                onClose={handleSortClose}
            />
            {/* Menu for selecting time created */}
            <TimeMenu
                anchorEl={timeAnchorEl}
                onClose={handleTimeClose}
                session={session}
            />
            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', marginBottom: 1 }}>
                <SiteSearchBar
                    id={`search-bar-${id}`}
                    placeholder={searchPlaceholder}
                    options={autocompleteOptions}
                    loading={loading}
                    value={searchString}
                    onChange={handleSearch}
                    onInputChange={onInputSelect}
                    session={session}
                    sxs={{ root: { width: 'min(100%, 600px)', paddingLeft: 2, paddingRight: 2 } }}
                />
            </Box>
            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', marginBottom: 1 }}>
                <Tooltip title="Sort by" placement="top">
                    <Box
                        onClick={handleSortOpen}
                        sx={searchButtonStyle(palette)}
                    >
                        <SortIcon fill={palette.secondary.main} />
                        <Typography variant="body2" sx={{ marginLeft: 0.5 }}>{sortByLabel}</Typography>
                    </Box>
                </Tooltip>
                <Tooltip title="Time created" placement="top">
                    <Box
                        onClick={handleTimeOpen}
                        sx={searchButtonStyle(palette)}
                    >
                        <TimeIcon fill={palette.secondary.main} />
                        <Typography variant="body2" sx={{ marginLeft: 0.5 }}>{timeFrameLabel}</Typography>
                    </Box>
                </Tooltip>
                {advancedSearchParams && <Tooltip title="See all search settings" placement="top">
                    <Box
                        onClick={handleAdvancedSearchDialogOpen}
                        sx={searchButtonStyle(palette)}
                    >
                        <BuildIcon fill={palette.secondary.main} />
                        {Object.keys(advancedSearchParams).length > 0 && <Typography variant="body2" sx={{ marginLeft: 0.5 }}>
                            *{Object.keys(advancedSearchParams).length}
                        </Typography>}
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
                <Button fullWidth onClick={handleAdd} startIcon={<PlusIcon />}>{t(`common:AddNew`, { lng: getUserLanguages(session)[0] })}</Button>
            </Box>}
        </>
    )
}