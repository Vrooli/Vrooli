/**
 * Search list for a single object type
 */
import { ListObject, NavigableObject, funcTrue } from "@local/shared";
import { Box } from "@mui/material";
import { SearchButtons } from "components/buttons/SearchButtons/SearchButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { SiteSearchBar } from "components/inputs/search";
import { useInfiniteScroll } from "hooks/gestures";
import { ReactNode, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ArgsType } from "types";
import { getDummyListLength } from "utils/consts";
import { openObject } from "utils/navigation/openObject";
import { ObjectList } from "../ObjectList/ObjectList";
import { ObjectListActions, SearchListProps } from "../types";

export function SearchList<DataType extends ListObject>({
    advancedSearchParams,
    advancedSearchSchema,
    allData,
    autocompleteOptions,
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
    removeItem,
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

    const handleSearch = useCallback(function handleSearchCallback(newString: string) {
        setSearchString(newString);
    }, [setSearchString]);

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

    const searchBarStyle = useMemo(function searchBarStyleMemo() {
        return {
            root: {
                margin: "auto",
                marginTop: 1,
                marginBottom: 1,
                width: "min(100%, 600px)",
                paddingLeft: 2,
                paddingRight: 2,
                ...sxs?.search,
                ...(variant === "minimal" ? { display: "none" } : {}),
            },
        } as const;
    }, [sxs?.search, variant]);

    const searchButtonsStyle = useMemo(function searchButtonsStyleMemo() {
        return {
            marginBottom: 2,
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
            <SiteSearchBar
                id={`${scrollContainerId}-search-bar`}
                placeholder={searchPlaceholder}
                options={autocompleteOptions}
                loading={loading}
                value={searchString}
                onChange={handleSearch}
                onInputChange={onInputSelect}
                sxs={searchBarStyle}
            />
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
                sx={searchButtonsStyle}
                timeFrame={timeFrame}
            />
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
