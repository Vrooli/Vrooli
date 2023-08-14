/**
 * Search list for a single object type
 */
import { Box, Button } from "@mui/material";
import { SearchButtons } from "components/buttons/SearchButtons/SearchButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { SiteSearchBar } from "components/inputs/search";
import { PlusIcon } from "icons";
import { useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { NavigableObject } from "types";
import { ListObject } from "utils/display/listTools";
import { useFindMany } from "utils/hooks/useFindMany";
import { openObject } from "utils/navigation/openObject";
import { ObjectList } from "../ObjectList/ObjectList";
import { SearchListProps } from "../types";

export function SearchList<DataType extends NavigableObject>({
    canNavigate = () => true,
    canSearch = () => true,
    dummyLength = 5,
    handleAdd,
    hideUpdateButton,
    id,
    searchPlaceholder,
    take = 20,
    resolve,
    searchType,
    sxs,
    onItemClick,
    onScrolledFar,
    where,
    zIndex,
}: SearchListProps) {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const {
        advancedSearchParams,
        advancedSearchSchema,
        allData,
        autocompleteOptions,
        loading,
        loadMore,
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
        resolve,
        searchType,
        take,
        where,
    });

    // If near the bottom of the page, load more data
    // If scrolled past a certain point, show an "Add New" button
    const handleScroll = useCallback(() => {
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

    const handleSearch = useCallback((newString: string) => { setSearchString(newString); }, [setSearchString]);

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

    return (
        <>
            <Box
                sx={{
                    display: "flex",
                    justifyContent: "center",
                    alignItems: "center",
                    marginBottom: 1,
                    ...(sxs?.search ?? {}),
                }}
            >
                <SiteSearchBar
                    id={`search-bar-${id}`}
                    placeholder={searchPlaceholder}
                    options={autocompleteOptions}
                    loading={loading}
                    value={searchString}
                    onChange={handleSearch}
                    onInputChange={onInputSelect}
                    sxs={{ root: { width: "min(100%, 600px)", paddingLeft: 2, paddingRight: 2 } }}
                    zIndex={zIndex}
                />
            </Box>
            <SearchButtons
                advancedSearchParams={advancedSearchParams}
                advancedSearchSchema={advancedSearchSchema}
                searchType={searchType}
                setAdvancedSearchParams={setAdvancedSearchParams}
                setSortBy={setSortBy}
                setTimeFrame={setTimeFrame}
                sortBy={sortBy}
                sortByOptions={sortByOptions}
                sx={{
                    marginBottom: 2,
                    ...sxs?.buttons,
                }}
                timeFrame={timeFrame}
                zIndex={zIndex}
            />
            <ListContainer
                emptyText={t("NoResults", { ns: "error" })}
                isEmpty={allData.length === 0 && !loading}
                sx={{ ...sxs?.listContainer }}
            >
                <ObjectList
                    canNavigate={canNavigate}
                    dummyItems={new Array(dummyLength).fill(searchType)}
                    hideUpdateButton={hideUpdateButton}
                    items={allData as ListObject[]}
                    keyPrefix={`${searchType}-list-item`}
                    loading={loading}
                    onClick={onItemClick}
                    zIndex={zIndex}
                />
            </ListContainer>
            {/* Add new button */}
            {Boolean(handleAdd) && <Box sx={{
                maxWidth: "400px",
                margin: "auto",
                paddingTop: 5,
            }}>
                <Button
                    fullWidth
                    onClick={handleAdd}
                    startIcon={<PlusIcon />}
                    variant="contained"
                >{t("AddNew")}</Button>
            </Box>}
        </>
    );
}
