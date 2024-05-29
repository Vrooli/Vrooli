/**
 * Search list for a single object type
 */
import { ListObject, NavigableObject } from "@local/shared";
import { Box, Button } from "@mui/material";
import { SearchButtons } from "components/buttons/SearchButtons/SearchButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { SiteSearchBar } from "components/inputs/search";
import { PlusIcon } from "icons";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ArgsType } from "types";
import { openObject } from "utils/navigation/openObject";
import { ObjectList } from "../ObjectList/ObjectList";
import { ObjectListActions, SearchListProps } from "../types";

export function SearchList<DataType extends ListObject>({
    advancedSearchParams,
    advancedSearchSchema,
    allData,
    autocompleteOptions,
    canNavigate = () => true,
    defaultSortBy,
    display,
    dummyLength = 5,
    handleAdd,
    handleToggleSelect,
    hideUpdateButton,
    id,
    isSelecting,
    loading,
    loadMore,
    onItemClick,
    removeItem,
    searchPlaceholder,
    searchString,
    searchType,
    selectedItems,
    setAdvancedSearchParams,
    setSortBy,
    setSearchString,
    setTimeFrame,
    sortBy,
    sortByOptions,
    sxs,
    timeFrame,
    updateItem,
}: SearchListProps<DataType>) {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    // Selected items which don't appear in allData (e.g. when searching)
    const selectedItemsNotInSearch = useMemo(() => {
        if (!selectedItems) return [];
        return selectedItems.filter(item => !allData.some(d => d.id === item.id));
    }, [allData, selectedItems]);

    const onAction = useCallback((action: keyof ObjectListActions<DataType>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted":
                removeItem(...(data as ArgsType<ObjectListActions<DataType>["Deleted"]>));
                break;
            case "Updated":
                updateItem(...(data as ArgsType<ObjectListActions<DataType>["Updated"]>));
                break;
        }
    }, [removeItem, updateItem]);

    // Handle infinite scroll
    const containerRef = useRef<HTMLDivElement>(null);
    const getScrollingContainer = useCallback((element: HTMLElement | null): HTMLElement | Document | null => {
        // If display is "page", use document instead
        if (display === "page") return document;
        // Traverse up the DOM
        while (element) {
            // If a dialog, find the first component with a role of "dialog", 
            if (display === "dialog" && element.getAttribute("role") === "dialog") {
                return element;
            }
            // If inline, find the first component with overflowY set to "scroll" or "auto"
            const overflowY = window.getComputedStyle(element).overflowY; //TODO need to fix this to get ChatSideMenu infinite scroll to work, but in a way that doesn't break FindObjectDialog
            if (display === "partial" && (overflowY === "scroll" || overflowY === "auto")) {
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
        // If in selection mode, toggle selection instead of navigating
        if (isSelecting) {
            typeof handleToggleSelect === "function" && handleToggleSelect(selectedItem);
            return;
        }
        // If onItemClick is supplied, call it instead of navigating
        if (typeof onItemClick === "function") {
            onItemClick(selectedItem);
            return;
        }
        // If canNavigate is supplied, call it
        if (canNavigate) {
            const shouldContinue = canNavigate(selectedItem);
            if (shouldContinue === false) return;
        }
        // Navigate to the object's page
        openObject(selectedItem as NavigableObject, setLocation);
    }, [allData, canNavigate, handleToggleSelect, isSelecting, onItemClick, setLocation]);

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
                controlsUrl={display === "page"}
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
                isEmpty={allData.length === 0 && selectedItemsNotInSearch.length === 0 && !loading}
                sx={{ ...sxs?.listContainer }}
            >
                <ObjectList
                    canNavigate={canNavigate}
                    dummyItems={new Array(dummyLength).fill(searchType)}
                    handleToggleSelect={handleToggleSelect}
                    hideUpdateButton={hideUpdateButton}
                    isSelecting={isSelecting}
                    items={[...selectedItemsNotInSearch, ...allData] as ListObject[]}
                    keyPrefix={`${searchType}-list-item`}
                    loading={loading}
                    onAction={onAction}
                    onClick={onItemClick}
                    selectedItems={selectedItems}
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
