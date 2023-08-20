/**
 * Search list for a single object type
 */
import { Box, Button } from "@mui/material";
import { SearchButtons } from "components/buttons/SearchButtons/SearchButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { SiteSearchBar } from "components/inputs/search";
import { useFindMany } from "hooks/useFindMany";
import { PlusIcon } from "icons";
import { useCallback, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { NavigableObject } from "types";
import { ListObject } from "utils/display/listTools";
import { openObject } from "utils/navigation/openObject";
import { ObjectList } from "../ObjectList/ObjectList";
import { SearchListProps } from "../types";

export function SearchList<DataType extends NavigableObject>({
    canNavigate = () => true,
    canSearch,
    display,
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
    where,
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

    // Handle infinite scroll
    const containerRef = useRef<HTMLDivElement>(null);
    const getScrollingContainer = useCallback((element: HTMLElement | null): HTMLElement | Document | null => {
        // If display is "page", use document instead
        if (display === "page") return document;
        // Otherwise, traverse up the DOM until we find a component with a role of "dialog"
        while (element) {
            if (element.getAttribute("role") === "dialog") {
                return element;
            }
            element = element.parentElement;
        }
        return null;
    }, [display]);
    const handleScroll = useCallback(() => {
        const container = getScrollingContainer(containerRef.current) ?? window;
        if (!container) return;
        let scrolledY: number;
        let scrollableHeight: number;
        if (container === document) {
            // When container is document, you should use document.documentElement or document.body based on browser compatibility
            scrolledY = window.scrollY || window.pageYOffset;
            scrollableHeight = document.documentElement.scrollHeight - window.innerHeight;
        } else if (container instanceof HTMLElement) {
            scrolledY = container.scrollTop;
            scrollableHeight = container.scrollHeight - container.clientHeight;
        } else {
            return;
        }
        if (!loading && scrolledY > scrollableHeight - 500) {
            loadMore();
        }
    }, [getScrollingContainer, loading, loadMore]);
    useEffect(() => {
        const scrollingContainer = getScrollingContainer(containerRef.current);
        if (scrollingContainer) {
            scrollingContainer.addEventListener("scroll", handleScroll);
            return () => scrollingContainer.removeEventListener("scroll", handleScroll);
        } else {
            console.error("Could not find scrolling container - infinite scroll disabled");
            return;
        }
    }, [getScrollingContainer, handleScroll]);

    const handleSearch = useCallback((newString: string) => {
        console.log("handleSearch called", newString);
        setSearchString(newString);
    }, [setSearchString]);

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
            />
            <ListContainer
                ref={containerRef}
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
