import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { PlusIcon } from "@local/icons";
import { Box, Button } from "@mui/material";
import { useCallback, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { listToListItems } from "../../../utils/display/listTools";
import { useFindMany } from "../../../utils/hooks/useFindMany";
import { openObject } from "../../../utils/navigation/openObject";
import { useLocation } from "../../../utils/route";
import { SearchButtons } from "../../buttons/SearchButtons/SearchButtons";
import { ListContainer } from "../../containers/ListContainer/ListContainer";
import { SiteSearchBar } from "../../inputs/search";
export function SearchList({ beforeNavigation, canSearch = true, handleAdd, hideUpdateButton, id, searchPlaceholder, take = 20, resolve, searchType, sxs, onScrolledFar, where, zIndex, }) {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { advancedSearchParams, advancedSearchSchema, allData, autocompleteOptions, loading, loadMore, pageData, parseData, searchString, setAdvancedSearchParams, setSortBy, setSearchString, setTimeFrame, sortBy, sortByOptions, timeFrame, } = useFindMany({
        canSearch,
        resolve,
        searchType,
        take,
        where,
    });
    const listItems = useMemo(() => listToListItems({
        beforeNavigation,
        dummyItems: new Array(5).fill(searchType),
        hideUpdateButton,
        items: (allData.length > 0 ? allData : parseData(pageData)),
        keyPrefix: `${searchType}-list-item`,
        loading,
        zIndex,
    }), [beforeNavigation, searchType, hideUpdateButton, allData, parseData, pageData, loading, zIndex]);
    const handleScroll = useCallback(() => {
        const scrolledY = window.scrollY;
        const windowHeight = window.innerHeight;
        if (!loading && scrolledY > windowHeight - 500) {
            loadMore();
        }
        if (scrolledY > 100) {
            if (onScrolledFar)
                onScrolledFar();
        }
    }, [loading, loadMore, onScrolledFar]);
    useEffect(() => {
        window.addEventListener("scroll", handleScroll);
        return () => window.removeEventListener("scroll", handleScroll);
    }, [handleScroll]);
    const handleSearch = useCallback((newString) => { setSearchString(newString); }, [setSearchString]);
    const onInputSelect = useCallback((newValue) => {
        if (!newValue)
            return;
        const selectedItem = allData.find(o => o?.id === newValue?.id);
        if (!selectedItem)
            return;
        openObject(selectedItem, setLocation);
    }, [allData, setLocation]);
    return (_jsxs(_Fragment, { children: [_jsx(Box, { sx: {
                    display: "flex",
                    justifyContent: "center",
                    alignItems: "center",
                    marginBottom: 1,
                    ...(sxs?.search ?? {}),
                }, children: _jsx(SiteSearchBar, { id: `search-bar-${id}`, placeholder: searchPlaceholder, options: autocompleteOptions, loading: loading, value: searchString, onChange: handleSearch, onInputChange: onInputSelect, sxs: { root: { width: "min(100%, 600px)", paddingLeft: 2, paddingRight: 2 } } }) }), _jsx(SearchButtons, { advancedSearchParams: advancedSearchParams, advancedSearchSchema: advancedSearchSchema, searchType: searchType, setAdvancedSearchParams: setAdvancedSearchParams, setSortBy: setSortBy, setTimeFrame: setTimeFrame, sortBy: sortBy, sortByOptions: sortByOptions, timeFrame: timeFrame, zIndex: zIndex }), _jsx(ListContainer, { emptyText: t("NoResults", { ns: "error" }), isEmpty: listItems.length === 0, children: listItems }), Boolean(handleAdd) && _jsx(Box, { sx: {
                    maxWidth: "400px",
                    margin: "auto",
                    paddingTop: 5,
                }, children: _jsx(Button, { fullWidth: true, onClick: handleAdd, startIcon: _jsx(PlusIcon, {}), children: t("AddNew") }) })] }));
}
//# sourceMappingURL=SearchList.js.map