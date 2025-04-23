/**
 * Search list for a single object type
 */
import { ListObject, NavigableObject, funcTrue } from "@local/shared";
import { Box } from "@mui/material";
import { ReactNode, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useInfiniteScroll } from "../../../hooks/gestures.js";
import { useLocation } from "../../../route/router.js";
import { ArgsType } from "../../../types.js";
import { getDummyListLength } from "../../../utils/consts.js";
import { openObject } from "../../../utils/navigation/openObject.js";
import { SearchButtons } from "../../buttons/SearchButtons.js";
import { ListContainer } from "../../containers/ListContainer.js";
import { BasicSearchBar, PaperSearchBar } from "../../inputs/search/SiteSearchBar.js";
import { ObjectList } from "../ObjectList/ObjectList.js";
import { ObjectListActions, SearchListProps } from "../types.js";

/**
 * Keys used for keyboard navigation in the search list
 */
const KEYBOARD_KEYS = {
    DOWN: "ArrowDown",
    UP: "ArrowUp",
    ENTER: "Enter",
    ESCAPE: "Escape",
    TAB: "Tab",
};

export function SearchList<DataType extends ListObject>({
    advancedSearchParams,
    advancedSearchSchema,
    allData,
    borderRadius,
    canNavigate = funcTrue,
    display,
    dummyLength,
    handleToggleSelect,
    hideUpdateButton,
    isSelecting,
    loading,
    loadMore,
    onItemClick,
    onSearchBlur,
    onSearchFocus,
    removeItem,
    searchBarVariant,
    scrollContainerId,
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
    variant,
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

    useInfiniteScroll({
        loading,
        loadMore,
        scrollContainerId,
    });

    /**
     * When an autocomplete item is selected, navigate to object
     */
    const onInputSelect = useCallback(function onInputSelectCallback(newValue: any) {
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

    /**
     * Handle keyboard navigation from search bar to results list
     */
    const onSearchKeyDown = useCallback((event: React.KeyboardEvent) => {
        if (event.key === KEYBOARD_KEYS.DOWN || (event.key === KEYBOARD_KEYS.TAB && !event.shiftKey)) {
            // When pressing down arrow or tab from search, focus the first result if available
            if (allData.length > 0) {
                event.preventDefault();
                const firstResultElement = document.querySelector(`#${scrollContainerId}-list [tabindex="0"]`) as HTMLElement;
                if (firstResultElement) {
                    firstResultElement.focus();
                }
            }
        }
    }, [allData.length, scrollContainerId]);

    const searchButtonsStyle = useMemo(function searchButtonsStyleMemo() {
        return {
            paddingTop: 2,
            paddingBottom: 2,
            ...sxs?.buttons,
            ...(variant === "minimal" ? { display: "none" } : {}),
        } as const;
    }, [sxs?.buttons, variant]);

    const listContainerStyle = useMemo(function listContainerStyleMemo() {
        return {
            overflow: "auto",
            ...sxs?.listContainer,
        } as const;
    }, [sxs?.listContainer]);

    return (
        <>
            <Box sx={sxs?.searchBarAndButtonsBox}>
                {searchBarVariant === "basic" && (
                    <BasicSearchBar
                        id={`${scrollContainerId}-search-bar`}
                        placeholder={searchPlaceholder}
                        value={searchString}
                        onChange={setSearchString}
                        onFocus={onSearchFocus}
                        onBlur={onSearchBlur}
                        onKeyDown={onSearchKeyDown}
                    />
                )}
                {searchBarVariant !== "basic" && (
                    <PaperSearchBar
                        id={`${scrollContainerId}-search-bar`}
                        placeholder={searchPlaceholder}
                        value={searchString}
                        onChange={setSearchString}
                        onFocus={onSearchFocus}
                        onBlur={onSearchBlur}
                        onKeyDown={onSearchKeyDown}
                    />
                )}
                <SearchButtons
                    advancedSearchParams={advancedSearchParams}
                    advancedSearchSchema={advancedSearchSchema}
                    controlsUrl={display === "Page"}
                    searchType={searchType}
                    setAdvancedSearchParams={setAdvancedSearchParams}
                    setSortBy={setSortBy}
                    setTimeFrame={setTimeFrame}
                    sortBy={sortBy}
                    sortByOptions={sortByOptions}
                    sx={searchButtonsStyle}
                    timeFrame={timeFrame}
                />
            </Box>
            <ListContainer
                id={`${scrollContainerId}-list`}
                borderRadius={borderRadius}
                emptyText={t("NoResults", { ns: "error" })}
                isEmpty={allData.length === 0 && selectedItemsNotInSearch.length === 0 && !loading}
                sx={listContainerStyle}
            >
                <ObjectList
                    canNavigate={canNavigate}
                    dummyItems={new Array(dummyLength || getDummyListLength(display)).fill(searchType)}
                    enableKeyboardNavigation={true}
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
        </>
    );
}

const searchListScrollContainerStyle = {
    height: "100%",
    overflowY: "auto",
} as const;

export function SearchListScrollContainer({
    children,
    id,
}: {
    children: ReactNode;
    id: string;
}) {
    return (
        <Box id={id} sx={searchListScrollContainerStyle}>
            {children}
        </Box>
    );
}
