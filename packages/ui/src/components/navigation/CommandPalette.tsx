import { ActionOption, ListObject, AutocompleteOption as SearchResult, ShortcutOption, VisibilityType, getObjectUrl } from "@local/shared";
import { Box, ButtonBase, DialogContent, Divider, List, ListItemButton, ListItemIcon, ListItemText, Typography, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../contexts/session.js";
import { useInfiniteScroll } from "../../hooks/gestures.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { useMenu } from "../../hooks/useMenu.js";
import { useLocation } from "../../route/router.js";
import { randomString } from "../../utils/codes.js";
import { ELEMENT_IDS, Z_INDEX } from "../../utils/consts.js";
import { normalizeText } from "../../utils/display/documentTools.js";
import { listToAutocomplete } from "../../utils/display/listTools.js";
import { getUserLanguages } from "../../utils/display/translationTools.js";
import { Actions, getAutocompleteOptionIcon, performAction, shortcuts } from "../../utils/navigation/quickActions.js";
import { MenuPayloads } from "../../utils/pubsub.js";
import { LargeDialog } from "../dialogs/LargeDialog/LargeDialog.js";
import { BasicSearchBar } from "../inputs/search/SiteSearchBar.js";

const titleId = "command-palette-dialog-title";
const scrollContainerId = "command-palette-search";
const searchBarId = `${scrollContainerId}-search-bar`;
const actionsListId = `${scrollContainerId}-actions-list`;
const shortcutsListId = `${scrollContainerId}-shortcuts-list`;
const searchListId = `${scrollContainerId}-search-list`;
const autoFocusDelayMs = 100;
const defaultItemsToShow = 5;
const dialogStyle = {
    root: {
        zIndex: Z_INDEX.CommandPalette,
    },
    paper: {
        margin: 0,
    },
} as const;
const where = {
    visibility: VisibilityType.OwnOrPublic,
} as const;

/**
 * Renders a section of search results with a title and optional "show more" button
 */
type ResultSectionProps = {
    title: string;
    items: SearchResult[];
    onSelect: (item: SearchResult) => void;
    id: string;
};

function ResultsSection({ title, items, onSelect, id }: ResultSectionProps) {
    const [showAll, setShowAll] = useState(false);
    const { t } = useTranslation();

    function toggleShowAll() {
        setShowAll(!showAll);
    }

    // Only show first 5 items unless "show all" is clicked
    const displayItems = showAll ? items : items.slice(0, defaultItemsToShow);
    const hasMore = items.length > defaultItemsToShow;

    if (items.length === 0) return null;

    return (
        <Box mb={2}>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={1} px={2}>
                <Typography variant="subtitle1" fontWeight="bold">{title}</Typography>
                {hasMore && (
                    <ButtonBase onClick={toggleShowAll}>
                        <Typography variant="body2" color="primary">
                            {showAll ? t("ShowLess") : t("ShowAll")}
                        </Typography>
                    </ButtonBase>
                )}
            </Box>
            <List id={id} dense>
                {displayItems.map(item => (
                    <ListItemButton key={item.key} onClick={() => onSelect(item)}>
                        <ListItemIcon>
                            {getAutocompleteOptionIcon(item.__typename, "currentColor")}
                        </ListItemIcon>
                        <ListItemText primary={item.label} />
                    </ListItemButton>
                ))}
            </List>
            <Divider />
        </Box>
    );
}

/**
 * Fancy search dialog for shortcuts, actions, public search, and private search.
 */
export function CommandPalette() {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    // Load all shortcuts in the user's language
    const shortcutsItems = useMemo<ShortcutOption[]>(() => shortcuts.map((shortcut) => {
        // Translate label and keywords
        const label = t(shortcut.label, { ...(shortcut.labelArgs ?? {}), defaultValue: shortcut.label }) as string;
        const keywords = shortcut.keywords?.map(keyword => {
            if (typeof keyword === "string") {
                return t(keyword) as string;
            }
            const { key, ...args } = keyword;
            return t(key, { ...args, defaultValue: key });
        });
        // Use the label and keywords to build search terms
        const searchTerms: string[] = [];
        searchTerms.push(normalizeText(label).trim().toLowerCase());
        for (const keyword of keywords ?? []) {
            searchTerms.push(normalizeText(keyword).trim().toLowerCase());
        }
        return {
            __typename: "Shortcut" as const,
            id: shortcut.value,
            keywords: searchTerms,
            label,
        } as const;
    }), [t]);

    // Load all actions
    const actionsItems = useMemo<ActionOption[]>(() => Object.values(Actions).map(action => {
        // Translate label and keywords
        const label = t(action.label, { ...(action.labelArgs ?? {}), defaultValue: action.label }) as string;
        const keywords = action.keywords?.map(keyword => {
            if (typeof keyword === "string") {
                return t(keyword) as string;
            }
            const { key, ...args } = keyword;
            return t(key, { ...args, defaultValue: key });
        });
        // Use the label and keywords to build search terms
        const searchTerms: string[] = [];
        searchTerms.push(normalizeText(label).trim().toLowerCase());
        for (const keyword of keywords ?? []) {
            searchTerms.push(normalizeText(keyword).trim().toLowerCase());
        }
        return {
            __typename: "Action" as const,
            canPerform: action.canPerform,
            id: action.id,
            keywords: searchTerms,
            label,
        } as const;
    }), [t]);

    // Fetch search results (also handles search string state)
    const findManyData = useFindMany<ListObject>({
        controlsUrl: false,
        searchType: "Popular" as const,
        take: 20,
        where,
    });
    useInfiniteScroll({
        loading: findManyData.loading,
        loadMore: findManyData.loadMore,
        scrollContainerId,
    });

    const updateSearch = useCallback((newValue: string) => {
        findManyData.setSearchString(newValue);
    }, [findManyData]);

    const hasResetData = useRef(false);
    const onEvent = useCallback(function onEventCallback({ isOpen }: MenuPayloads[typeof ELEMENT_IDS.CommandPalette]) {
        // If the command palette is opened
        if (isOpen) {
            // Focus the search bar
            setTimeout(() => {
                const inputElement = document.getElementById(searchBarId);
                if (inputElement) {
                    inputElement.focus();
                }
            }, autoFocusDelayMs);
            // Reset the hasResetData flag
            hasResetData.current = false;
        }
        // If the command palette is closed
        else {
            // Clear the search string
            findManyData.setSearchString("");
            // Clear the search result
            if (findManyData.allData.length > 0 && !hasResetData.current) {
                findManyData.setAllData([]);
                hasResetData.current = true;
            }
        }
    }, [findManyData]);
    const { isOpen, close } = useMenu({
        id: ELEMENT_IDS.CommandPalette,
        onEvent,
    });

    // Get filtered results by category
    const { filteredActions, filteredShortcuts, searchResultsData } = useMemo(() => {
        const filterString = normalizeText(findManyData.searchString).trim().toLowerCase();
        // Create unique keys for each option
        const keySet = new Set<string>();

        function addKey<T extends { id: string }>(option: T) {
            const key = option.id;
            // If the key is already in the set, add a random string to the id. 
            // This ensures that key collisions are rare, since they can break the UI in unexpected ways.
            while (keySet.has(key)) {
                console.warn("Duplicate key found in command palette", key, option);
                option.id += `-${randomString()}`;
            }
            keySet.add(option.id);
            return { ...option, key } as T & { key: string };
        }

        // Filter actions by search string
        const filteredActions = actionsItems
            .filter(action => action.keywords.some(keyword => keyword.includes(filterString)))
            .map(addKey);

        // Filter shortcuts by search string
        const filteredShortcuts = shortcutsItems
            .filter(shortcut => shortcut.keywords.some(keyword => keyword.includes(filterString)))
            .map(addKey);

        // Get search results
        const searchResultsData = listToAutocomplete(findManyData.allData, getUserLanguages(session))
            .map(addKey);

        return { filteredActions, filteredShortcuts, searchResultsData };
    }, [actionsItems, findManyData.allData, findManyData.searchString, session, shortcutsItems]);

    /**
     * When an autocomplete item is selected, navigate to object
     */
    const onInputSelect = useCallback(function onInputSelectCallback(newValue: SearchResult) {
        if (!newValue) return;
        // Clear search string and close command palette
        close();
        findManyData.setSearchString("");
        // If the option is an action, perform it
        if (newValue.__typename === "Action") {
            performAction(newValue, session);
            return;
        }
        // Get object url
        const newLocation = getObjectUrl(newValue);
        if (!newLocation) return;
        // Set new location
        setLocation(newLocation);
    }, [close, findManyData, session, setLocation]);

    const searchBarBoxStyle = useMemo(function searchBarBoxStyleMemo() {
        return {
            background: palette.background.default,
            overflowY: "auto",
            minHeight: "500px",
            height: "100%",
        } as const;
    }, [palette]);

    const noResultsFound = filteredActions.length === 0 && filteredShortcuts.length === 0 && searchResultsData.length === 0;

    return (
        <LargeDialog
            id="command-palette-dialog"
            isOpen={isOpen}
            onClose={close}
            titleId={titleId}
            sxs={dialogStyle}
        >
            <BasicSearchBar
                id={searchBarId}
                onChange={updateSearch}
                placeholder={t("CommandPalettePlaceholder")}
                value={findManyData.searchString}
            />
            <DialogContent id={scrollContainerId} sx={searchBarBoxStyle}>
                {noResultsFound ? (
                    <Box display="flex" justifyContent="center" alignItems="center" height="100%">
                        <Typography variant="body1" color="text.secondary">{t("NoResults")}</Typography>
                    </Box>
                ) : (
                    <>
                        <ResultsSection
                            title={t("Action", { count: filteredActions.length })}
                            items={filteredActions}
                            onSelect={onInputSelect}
                            id={actionsListId}
                        />
                        <ResultsSection
                            title={t("Shortcut_one", { count: filteredShortcuts.length })}
                            items={filteredShortcuts}
                            onSelect={onInputSelect}
                            id={shortcutsListId}
                        />
                        <ResultsSection
                            title={t("SearchResult_other", { count: searchResultsData.length })}
                            items={searchResultsData}
                            onSelect={onInputSelect}
                            id={searchListId}
                        />
                    </>
                )}
            </DialogContent>
        </LargeDialog>
    );
}
