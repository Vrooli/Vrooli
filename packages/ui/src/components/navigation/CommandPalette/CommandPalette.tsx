import { endpointGetFeedPopular, LINKS, PopularInput, PopularResult, useLocation, uuidValidate } from "@local/shared";
import { DialogContent, useTheme } from "@mui/material";
import { DialogTitle } from "components/dialogs/DialogTitle/DialogTitle";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SiteSearchBar } from "components/inputs/search";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { AutocompleteOption, ShortcutOption } from "types";
import { listToAutocomplete } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { useDisplayServerError } from "utils/hooks/useDisplayServerError";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { getObjectUrl } from "utils/navigation/openObject";
import { actionsItems, shortcuts } from "utils/navigation/quickActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";

/**
 * Strips URL for comparison against the current URL.
 * @param url URL to strip
 * @returns Stripped URL
 */
const stripUrl = (url: string) => {
    // Split by '/' and remove empty strings
    const urlParts = new URL(url).pathname.split("/").filter(Boolean);
    // If last part is a UUID, or equal to "add" or "edit", remove it
    // For example, navigating from viewing the graph of an existing routine 
    // to creating a new multi-step routine (/routine/1234?build=true -> /routine/add?build=true) 
    // requires a reload
    if (urlParts.length > 1 &&
        (uuidValidate(urlParts[urlParts.length - 1]) ||
            urlParts[urlParts.length - 1] === "add" ||
            urlParts[urlParts.length - 1] === "edit")) {
        urlParts.pop();
    }
    return urlParts.join("/");
};

const titleId = "command-palette-dialog-title";

export const CommandPalette = () => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const languages = useMemo(() => getUserLanguages(session), [session]);

    const [searchString, setSearchString] = useState<string>("");
    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue); }, []);

    const [open, setOpen] = useState(false);
    const close = useCallback(() => setOpen(false), []);

    useEffect(() => {
        const dialogSub = PubSub.get().subscribeCommandPalette(() => {
            setOpen(o => !o);
            setSearchString("");
        });
        return () => { PubSub.get().unsubscribe(dialogSub); };
    }, []);

    const [refetch, { data, loading, errors }] = useLazyFetch<PopularInput, PopularResult>({
        ...endpointGetFeedPopular,
        inputs: { searchString: searchString.replaceAll(/![^\s]{1,}/g, "") },
    });
    useEffect(() => { open && refetch(); }, [open, refetch, searchString]);
    useDisplayServerError(errors);

    const shortcutsItems = useMemo<ShortcutOption[]>(() => shortcuts.map(({ label, labelArgs, value }) => ({
        __typename: "Shortcut",
        label: t(label, { ...(labelArgs ?? {}), defaultValue: label }) as string,
        id: value,
    })), [t]);

    const autocompleteOptions: AutocompleteOption[] = useMemo(() => {
        const firstResults: AutocompleteOption[] = [];
        // If "help" typed (or your language's equivalent), add help and faq shortcuts as first result
        const lowercaseHelp = t("Help").toLowerCase();
        if (searchString.toLowerCase().startsWith(lowercaseHelp)) {
            firstResults.push({
                __typename: "Shortcut",
                label: t("ShortcutBeginnersGuide"),
                id: LINKS.Welcome,
            }, {
                __typename: "Shortcut",
                label: t("ShortcutFaq"),
                id: LINKS.FAQ,
            });
        }
        // Group all query results and sort by number of bookmarks. Ignore any value that isn't an array
        const flattened = (Object.values(data ?? [])).filter(Array.isArray).reduce((acc, curr) => acc.concat(curr), []);
        const queryItems = listToAutocomplete(flattened, languages).sort((a: any, b: any) => {
            return b.bookmarks - a.bookmarks;
        });
        return [...firstResults, ...queryItems, ...shortcutsItems, ...actionsItems];
    }, [t, searchString, data, languages, shortcutsItems]);

    /**
     * When an autocomplete item is selected, navigate to object
     */
    const onInputSelect = useCallback((newValue: AutocompleteOption) => {
        if (!newValue) return;
        // Clear search string and close command palette
        close();
        setSearchString("");
        // Get object url
        // NOTE: actions don't require navigation, so they are ignored. 
        // The search bar performs the action automatically
        const newLocation = getObjectUrl(newValue);
        if (!newLocation) return;
        // If new pathname is the same, reload page
        const shouldReload = stripUrl(`${window.location.origin}${newLocation}`) === stripUrl(window.location.href);
        // Set new location
        setLocation(newLocation);
        if (shouldReload) window.location.reload();
    }, [close, setLocation]);

    return (
        <LargeDialog
            id="command-palette-dialog"
            isOpen={open}
            onClose={close}
            titleId={titleId}
            zIndex={10000}
        >
            <DialogTitle
                id={titleId}
                helpText={t("CommandPaletteHelp")}
                title={t("CommandPaletteTitle")}
                onClose={close}
            />
            <DialogContent sx={{
                background: palette.background.default,
                overflowY: "visible",
                minHeight: "500px",
            }}>
                <SiteSearchBar
                    id="command-palette-search"
                    autoFocus={true}
                    placeholder='CommandPalettePlaceholder'
                    options={autocompleteOptions}
                    loading={loading}
                    value={searchString}
                    onChange={updateSearch}
                    onInputChange={onInputSelect}
                    showSecondaryLabel={true}
                    sxs={{
                        root: {
                            width: "100%",
                            top: 0,
                            marginTop: 2,
                        },
                        paper: { background: palette.background.paper },
                    }}
                    zIndex={1000}
                />
            </DialogContent>
        </LargeDialog>
    );
};
