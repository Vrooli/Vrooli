import { useQuery } from "@apollo/client";
import { Box, Button, CircularProgress, List, Tooltip, Typography } from "@mui/material";
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
import { AutocompleteListItem, getListItemLabel, getUserLanguages, listToAutocomplete, listToListItems, Pubs } from "utils";
import { ListOrganization, ListProject, ListRoutine, ListStandard } from "types";

type ListItem = ListOrganization | ListProject | ListRoutine | ListStandard | ListOrganization;

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
        after.current = undefined;
        fetchPage();
    }, [searchString, sortBy, createdTimeFrame, fetchPage]);

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
    }, [pageData]);

    const autocompleteOptions: AutocompleteListItem[] = useMemo(() => {
        return listToAutocomplete(allData, getUserLanguages(session)).sort((a: any, b: any) => {
            return b.stars - a.stars;
        });
    }, [allData, session]);

    const listItems = useMemo(() => listToListItems(
        allData,
        session,
        itemKeyPrefix,
        (item) => onObjectSelect(item),
    ), [allData, session, itemKeyPrefix, onObjectSelect])

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
                    background: (t) => t.palette.background.paper,
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

    // Handle advanced search dialog
    const [advancedSearchDialogOpen, setAdvancedSearchDialogOpen] = useState<boolean>(false);
    const handleAdvancedSearchDialogOpen = useCallback(() => {
        // PubSub.publish(Pubs.Snack, { message: 'Available next update. Please be patient with us😬', severity: 'error' });
        // return;
        setAdvancedSearchDialogOpen(true)
    }, []);
    const handleAdvancedSearchDialogClose = useCallback(() => { setAdvancedSearchDialogOpen(false) }, []);

    return (
        <>
            {/* Dialog for setting advanced search items */}
            <AdvancedSearchDialog
                handleClose={handleAdvancedSearchDialogClose}
                handleSearch={() => { }}
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
                        <SortListIcon sx={{ fill: (t) => t.palette.secondary.main }} />
                        {sortByLabel}
                    </Box>
                </Tooltip>
                <Tooltip title="Time created" placement="top">
                    <Box
                        onClick={handleTimeOpen}
                        sx={{ ...searchButtonStyle }}
                    >
                        <TimeIcon sx={{ fill: (t) => t.palette.secondary.main }} />
                        {timeFrameLabel}
                    </Box>
                </Tooltip>
                <Tooltip title="See all search settings" placement="top">
                    <Box
                        onClick={handleAdvancedSearchDialogOpen}
                        sx={{ ...searchButtonStyle }}
                    >
                        <AdvancedIcon sx={{ fill: (t) => t.palette.secondary.main }} />
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