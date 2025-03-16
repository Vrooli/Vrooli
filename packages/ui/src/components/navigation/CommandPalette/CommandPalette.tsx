import { ActionOption, ListObject, AutocompleteOption as SearchResult, ShortcutOption, VisibilityType, getObjectUrl } from "@local/shared";
import { Box, DialogContent, List, ListItemButton, ListItemIcon, ListItemText, Typography, useTheme } from "@mui/material";
import { useInfiniteScroll } from "hooks/gestures.js";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { listToAutocomplete } from "utils/display/listTools.js";
import { getUserLanguages } from "utils/display/translationTools.js";
import { SessionContext } from "../../../contexts.js";
import { useFindMany } from "../../../hooks/useFindMany.js";
import { useLocation } from "../../../route/router.js";
import { randomString } from "../../../utils/codes.js";
import { Z_INDEX } from "../../../utils/consts.js";
import { normalizeText } from "../../../utils/display/documentTools.js";
import { Actions, getAutocompleteOptionIcon, performAction, shortcuts } from "../../../utils/navigation/quickActions.js";
import { PubSub } from "../../../utils/pubsub.js";
import { LargeDialog } from "../../dialogs/LargeDialog/LargeDialog.js";
import { BasicSearchBar } from "../../inputs/search/SiteSearchBar.js";

const titleId = "command-palette-dialog-title";
const scrollContainerId = "command-palette-search";
const searchBarId = `${scrollContainerId}-search-bar`;
const listId = `${scrollContainerId}-list`;
const autoFocusDelayMs = 100;
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
 * Fancy search dialog for shortcuts, actions, public search, and private search.
 */
export function CommandPalette() {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const [open, setOpen] = useState(false);
    const close = useCallback(() => setOpen(false), []);

    useEffect(function focusSearchBar() {
        if (open) {
            setTimeout(() => {
                const inputElement = document.getElementById(searchBarId);
                if (inputElement) {
                    inputElement.focus();
                }
            }, autoFocusDelayMs);
        }
    }, [open]);

    // Load all shortcuts in the user's language
    const shortcutsItems = useMemo<ShortcutOption[]>(() => shortcuts.map((shortcut) => {
        // Translate label and keywords
        const label = t(shortcut.label, { ...(shortcut.labelArgs ?? {}), defaultValue: shortcut.label }) as string;
        const keywords = shortcut.keywords?.map(keyword => t(keyword, { ...(keyword.labelArgs ?? {}), defaultValue: keyword.label }) as string);
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
        const keywords = action.keywords?.map(keyword => t(keyword, { ...(keyword.labelArgs ?? {}), defaultValue: keyword.label }) as string);
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
    const hasResetData = useRef(false);
    useEffect(function clearDataWhenClosed() {
        if (!open && findManyData.allData.length > 0 && !hasResetData.current) {
            findManyData.setAllData([]);
            hasResetData.current = true;
        } else if (open) {
            hasResetData.current = false;
        }
    }, [open, findManyData]);

    const updateSearch = useCallback((newValue: string) => {
        findManyData.setSearchString(newValue);
    }, [findManyData]);

    useEffect(function subscribeToCommandPalette() {
        const unsubscribe = PubSub.get().subscribe("commandPalette", () => {
            setOpen(o => !o);
            findManyData.setSearchString("");
        });
        return unsubscribe;
    }, [findManyData]);

    // Combine all results
    const searchResults: SearchResult[] = useMemo(() => {
        const filterString = normalizeText(findManyData.searchString).trim().toLowerCase();
        // Create unique keys for each option
        const keySet = new Set<string>();

        function addKey(option: Omit<SearchResult, "key">) {
            const key = option.id;
            // If the key is already in the set, add a random string to the id. 
            // This ensures that key collisions are rare, since they can break the UI in unexpected ways.
            while (keySet.has(key)) {
                console.warn("Duplicate key found in command palette", key, option);
                option.id += `-${randomString()}`;
            }
            keySet.add(option.id);
            return { ...option, key };
        }

        // Filter actions by search string
        const filteredActions = actionsItems
            .filter(action => action.keywords.some(keyword => keyword.includes(filterString)))
            .map(addKey);

        // Filter shortcuts by search string
        const filteredShortcuts = shortcutsItems
            .filter(shortcut => shortcut.keywords.some(keyword => keyword.includes(filterString)))
            .map(addKey);

        const fetchedData = listToAutocomplete(findManyData.allData, getUserLanguages(session))
            .map(addKey);

        // Combine all results
        const combinedOptions = [...filteredActions, ...filteredShortcuts, ...fetchedData] as SearchResult[];

        // Return the results
        return combinedOptions;
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

    return (
        <LargeDialog
            id="command-palette-dialog"
            isOpen={open}
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
            <DialogContent sx={searchBarBoxStyle}>
                {searchResults.length === 0 ? (
                    <Box display="flex" justifyContent="center" alignItems="center" height="100%">
                        <Typography variant="body1" color="text.secondary">No Results</Typography>
                    </Box>
                ) : (
                    <List id={listId}>
                        {searchResults.map(result => {
                            function handleClick() {
                                onInputSelect(result);
                            }
                            return (
                                <ListItemButton key={result.key} onClick={handleClick}>
                                    <ListItemIcon>
                                        {getAutocompleteOptionIcon(result.__typename, "currentColor")}
                                    </ListItemIcon>
                                    <ListItemText primary={result.label} />
                                </ListItemButton>
                            );
                        })}
                    </List>
                )}
            </DialogContent>
        </LargeDialog>
    );
}
