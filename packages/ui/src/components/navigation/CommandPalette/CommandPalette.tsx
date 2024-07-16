import { AutocompleteOption, endpointGetFeedPopular, getObjectUrl, PopularSearchInput, PopularSearchResult, ShortcutOption } from "@local/shared";
import { DialogContent, useTheme } from "@mui/material";
import { DialogTitle } from "components/dialogs/DialogTitle/DialogTitle";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SiteSearchBar } from "components/inputs/search";
import { SessionContext } from "contexts/SessionContext";
import { parseData } from "hooks/useFindMany";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { randomString } from "utils/codes";
import { listToAutocomplete } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { actionsItems, shortcuts } from "utils/navigation/quickActions";
import { PubSub } from "utils/pubsub";

const titleId = "command-palette-dialog-title";

export function CommandPalette() {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const languages = useMemo(() => getUserLanguages(session), [session]);

    const [searchString, setSearchString] = useState<string>("");
    const updateSearch = useCallback((newValue: string) => { setSearchString(newValue); }, []);

    const [open, setOpen] = useState(false);
    const close = useCallback(() => setOpen(false), []);

    useEffect(() => {
        const unsubscribe = PubSub.get().subscribe("commandPalette", () => {
            setOpen(o => !o);
            setSearchString("");
        });
        return unsubscribe;
    }, []);

    const [refetch, { data, loading }] = useLazyFetch<PopularSearchInput, PopularSearchResult>({
        ...endpointGetFeedPopular,
        inputs: { searchString: searchString.replace(/![^\s]{1,}/g, "") },
    });
    useEffect(() => { open && refetch(); }, [open, refetch, searchString]);

    const shortcutsItems = useMemo<ShortcutOption[]>(() => shortcuts.map(({ label, labelArgs, value }) => ({
        __typename: "Shortcut",
        label: t(label, { ...(labelArgs ?? {}), defaultValue: label }) as string,
        id: value,
    })), [t]);

    const autocompleteOptions: AutocompleteOption[] = useMemo(() => {
        const firstResults: AutocompleteOption[] = [];
        // If "help" typed (or your language's equivalent)
        const lowercaseHelp = t("Help").toLowerCase();
        if (searchString.toLowerCase().startsWith(lowercaseHelp)) {
            // firstResults.push({ TODO
            //     __typename: "Shortcut",
            //     label: t("Tutorial"),
            //     id: LINKS.Tutorial,
            // });
        }
        const queryItems = listToAutocomplete(parseData(data), languages);
        const combinedOptions = [...firstResults, ...queryItems, ...shortcutsItems, ...actionsItems];
        // Create unique keys for each option
        const keySet = new Set<string>();
        return combinedOptions.map(option => {
            const key = option.id;
            while (keySet.has(key)) {
                console.warn("Duplicate key found in command palette", key, option);
                option.id += `-${randomString()}`;
            }
            keySet.add(option.id);
            return { ...option, key };
        });
    }, [t, searchString, data, languages, shortcutsItems]);

    /**
     * When an autocomplete item is selected, navigate to object
     */
    const onInputSelect = useCallback(function onInputSelectCallback(newValue: AutocompleteOption) {
        if (!newValue) return;
        // Clear search string and close command palette
        close();
        setSearchString("");
        // Get object url
        const newLocation = getObjectUrl(newValue);
        if (!newLocation) return;
        // Set new location
        setLocation(newLocation);
    }, [close, setLocation]);

    const searchBarBoxStyle = useMemo(function searchBarBoxStyleMemo() {
        return {
            background: palette.background.default,
            overflowY: "visible",
            minHeight: "500px",
        } as const;
    }, [palette]);

    const searchBarStyle = useMemo(function searchBarStyleMemo() {
        return {
            root: {
                width: "100%",
                top: 0,
                marginTop: 2,
            },
            paper: { background: palette.background.paper },
        } as const;
    }, [palette]);

    return (
        <LargeDialog
            id="command-palette-dialog"
            isOpen={open}
            onClose={close}
            titleId={titleId}
        >
            <DialogTitle
                id={titleId}
                help={t("CommandPaletteHelp")}
                title={t("CommandPaletteTitle")}
                onClose={close}
            />
            <DialogContent sx={searchBarBoxStyle}>
                <SiteSearchBar
                    id="command-palette-search"
                    autoFocus={true}
                    placeholder='CommandPalettePlaceholder'
                    options={autocompleteOptions}
                    loading={loading}
                    value={searchString}
                    onChange={updateSearch}
                    onInputChange={onInputSelect}
                    sxs={searchBarStyle}
                />
            </DialogContent>
        </LargeDialog>
    );
}
