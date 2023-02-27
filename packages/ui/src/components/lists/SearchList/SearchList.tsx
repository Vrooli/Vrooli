/**
 * Search list for a single object type
 */
import { Box, Button, List, Typography, useTheme } from "@mui/material";
import { SearchButtonsList, SiteSearchBar } from "components";
import { useCallback, useEffect, useMemo } from "react";
import { PlusIcon } from '@shared/icons';
import { SearchListProps } from "../types";
import { listToListItems, openObject, useFindMany } from "utils";
import { useLocation } from '@shared/route';
import { useTranslation } from "react-i18next";
import { NavigableObject } from "types";

export function SearchList<DataType extends NavigableObject>({
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

    const {
        advancedSearchParams,
        advancedSearchSchema,
        allData,
        autocompleteOptions,
        loading,
        loadMore,
        pageData,
        parseData,
        searchString,
        setAdvancedSearchParams,
        setSortBy,
        setSearchString,
        setTimeFrame,
        sortBy,
        sortByOptions,
        timeFrame,
    } = useFindMany<DataType>({ 
        canSearch,
        searchType,
        session,
        take,
        where,
    });

    const listItems = useMemo(() => listToListItems({
        beforeNavigation,
        dummyItems: new Array(5).fill(searchType),
        hideRoles,
        items: (allData.length > 0 ? allData : parseData(pageData)) as any[],
        keyPrefix: `${searchType}-list-item`,
        loading,
        session: session,
        zIndex,
    }), [beforeNavigation, searchType, hideRoles, allData, parseData, pageData, loading, session, zIndex])

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
                    ) : (<Typography variant="h6" textAlign="center">{t(`NoResults`, { ns: 'error' })}</Typography>)
                }
            </Box>
        )
    }, [listItems, palette.background.paper, t]);


    return (
        <>
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
            <SearchButtonsList
                advancedSearchParams={advancedSearchParams}
                advancedSearchSchema={advancedSearchSchema}
                searchType={searchType}
                session={session}
                setAdvancedSearchParams={setAdvancedSearchParams}
                setSortBy={setSortBy}
                setTimeFrame={setTimeFrame}
                sortBy={sortBy}
                sortByOptions={sortByOptions}
                timeFrame={timeFrame}
                zIndex={zIndex}
            />
            {searchResultContainer}
            {/* Add new button */}
            {Boolean(handleAdd) && <Box sx={{
                maxWidth: '400px',
                margin: 'auto',
                paddingTop: 5,
            }}>
                <Button fullWidth onClick={handleAdd} startIcon={<PlusIcon />}>{t(`AddNew`)}</Button>
            </Box>}
        </>
    )
}