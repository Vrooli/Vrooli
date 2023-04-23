import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { uuidValidate } from "@local/uuid";
import { DialogContent, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { feedPopular } from "../../../api/generated/endpoints/feed_popular";
import { useCustomLazyQuery } from "../../../api/hooks";
import { listToAutocomplete } from "../../../utils/display/listTools";
import { getUserLanguages } from "../../../utils/display/translationTools";
import { useDisplayApolloError } from "../../../utils/hooks/useDisplayApolloError";
import { getObjectUrl } from "../../../utils/navigation/openObject";
import { actionsItems, shortcuts } from "../../../utils/navigation/quickActions";
import { PubSub } from "../../../utils/pubsub";
import { useLocation } from "../../../utils/route";
import { SessionContext } from "../../../utils/SessionContext";
import { DialogTitle } from "../../dialogs/DialogTitle/DialogTitle";
import { LargeDialog } from "../../dialogs/LargeDialog/LargeDialog";
import { SiteSearchBar } from "../../inputs/search";
const stripUrl = (url) => {
    const urlParts = new URL(url).pathname.split("/").filter(Boolean);
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
    const [searchString, setSearchString] = useState("");
    const updateSearch = useCallback((newValue) => { setSearchString(newValue); }, []);
    const [open, setOpen] = useState(false);
    const close = useCallback(() => setOpen(false), []);
    useEffect(() => {
        const dialogSub = PubSub.get().subscribeCommandPalette(() => {
            setOpen(o => !o);
            setSearchString("");
        });
        return () => { PubSub.get().unsubscribe(dialogSub); };
    }, []);
    const [refetch, { data, loading, error }] = useCustomLazyQuery(feedPopular, {
        variables: { searchString: searchString.replace(/![^\s]{1,}/g, "") },
        errorPolicy: "all",
    });
    useEffect(() => { open && refetch(); }, [open, refetch, searchString]);
    useDisplayApolloError(error);
    const shortcutsItems = useMemo(() => shortcuts.map(({ label, labelArgs, value }) => ({
        __typename: "Shortcut",
        label: t(label, { ...(labelArgs ?? {}), defaultValue: label }),
        id: value,
    })), [t]);
    const autocompleteOptions = useMemo(() => {
        const firstResults = [];
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
        const flattened = (Object.values(data ?? [])).filter(Array.isArray).reduce((acc, curr) => acc.concat(curr), []);
        const queryItems = listToAutocomplete(flattened, languages).sort((a, b) => {
            return b.bookmarks - a.bookmarks;
        });
        return [...firstResults, ...queryItems, ...shortcutsItems, ...actionsItems];
    }, [t, searchString, data, languages, shortcutsItems]);
    const onInputSelect = useCallback((newValue) => {
        if (!newValue)
            return;
        close();
        setSearchString("");
        const newLocation = getObjectUrl(newValue);
        if (!Boolean(newLocation))
            return;
        const shouldReload = stripUrl(`${window.location.origin}${newLocation}`) === stripUrl(window.location.href);
        setLocation(newLocation);
        if (shouldReload)
            window.location.reload();
    }, [close, setLocation]);
    return (_jsxs(LargeDialog, { id: "command-palette-dialog", isOpen: open, onClose: close, titleId: titleId, zIndex: 10000, children: [_jsx(DialogTitle, { id: titleId, helpText: t("CommandPaletteHelp"), title: t("CommandPaletteTitle"), onClose: close }), _jsx(DialogContent, { sx: {
                    background: palette.background.default,
                    overflowY: "visible",
                    minHeight: "500px",
                }, children: _jsx(SiteSearchBar, { id: "command-palette-search", autoFocus: true, placeholder: 'CommandPalettePlaceholder', options: autocompleteOptions, loading: loading, value: searchString, onChange: updateSearch, onInputChange: onInputSelect, showSecondaryLabel: true, sxs: {
                        root: {
                            width: "100%",
                            top: 0,
                            marginTop: 2,
                        },
                        paper: { background: palette.background.paper },
                    } }) })] }));
};
//# sourceMappingURL=CommandPalette.js.map