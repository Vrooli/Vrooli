import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    Dialog,
    DialogContent,
    useTheme,
} from '@mui/material';
import { actionsItems, getObjectUrl, getUserLanguages, listToAutocomplete, PubSub, shortcutsItems, useDisplayApolloError } from 'utils';
import { SiteSearchBar } from 'components/inputs';
import { APP_LINKS, PopularInput, PopularResult } from '@shared/consts';
import { AutocompleteOption } from 'types';
import { useCustomLazyQuery } from 'api/hooks';
import { CommandPaletteProps } from '../types';
import { useLocation } from '@shared/route';
import { DialogTitle } from 'components';
import { uuidValidate } from '@shared/uuid';
import { useTranslation } from 'react-i18next';
import { feedPopular } from 'api/generated/endpoints/feed_popular';

/**
 * Strips URL for comparison against the current URL.
 * @param url URL to strip
 * @returns Stripped URL
 */
const stripUrl = (url: string) => {
    // Split by '/' and remove empty strings
    let urlParts = new URL(url).pathname.split('/').filter(Boolean);
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
    return urlParts.join('/');
}

const titleId = 'command-palette-dialog-title';

export const CommandPalette = ({
    session
}: CommandPaletteProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const languages = useMemo(() => getUserLanguages(session), [session]);

    const [searchString, setSearchString] = useState<string>('');
    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue) }, []);

    const [open, setOpen] = useState(false);
    const close = useCallback(() => setOpen(false), []);

    useEffect(() => {
        let dialogSub = PubSub.get().subscribeCommandPalette(() => {
            setOpen(o => !o);
            setSearchString('');
        });
        return () => { PubSub.get().unsubscribe(dialogSub) };
    }, [])

    const [refetch, { data, loading, error }] = useCustomLazyQuery<PopularResult, PopularInput>(feedPopular, {
        variables: { searchString: searchString.replaceAll(/![^\s]{1,}/g, '') },
        errorPolicy: 'all'
    });
    useEffect(() => { open && refetch() }, [open, refetch, searchString]);
    useDisplayApolloError(error);


    const autocompleteOptions: AutocompleteOption[] = useMemo(() => {
        const firstResults: AutocompleteOption[] = [];
        // If "help" typed (or your language's equivalent), add help and faq shortcuts as first result
        const lowercaseHelp = t(`Help`).toLowerCase();
        if (searchString.toLowerCase().startsWith(lowercaseHelp)) {
            firstResults.push({
                __typename: "Shortcut",
                label: t(`ShortcutBeginnersGuide`),
                id: APP_LINKS.Welcome,
            }, {
                __typename: "Shortcut",
                label: t(`ShortcutFaq`),
                id: APP_LINKS.FAQ,
            });
        }
        // Group all query results and sort by number of bookmarks. Ignore any value that isn't an array
        const flattened = (Object.values(data ?? [])).filter(Array.isArray).reduce((acc, curr) => acc.concat(curr), []);
        const queryItems = listToAutocomplete(flattened, languages).sort((a: any, b: any) => {
            return b.bookmarks - a.bookmarks;
        });
        return [...firstResults, ...queryItems, ...shortcutsItems, ...actionsItems];
    }, [searchString, data, languages, t]);

    /**
     * When an autocomplete item is selected, navigate to object
     */
    const onInputSelect = useCallback((newValue: AutocompleteOption) => {
        if (!newValue) return;
        // Clear search string and close command palette
        close();
        setSearchString('');
        // Get object url
        // NOTE: actions don't require navigation, so they are ignored. 
        // The search bar performs the action automatically
        const newLocation = getObjectUrl(newValue);
        if (!Boolean(newLocation)) return;
        // If new pathname is the same, reload page
        const shouldReload = stripUrl(`${window.location.origin}${newLocation}`) === stripUrl(window.location.href);
        // Set new location
        setLocation(newLocation);
        if (shouldReload) window.location.reload();
    }, [close, setLocation]);

    return (
        <Dialog
            open={open}
            onClose={close}
            aria-labelledby={titleId}
            sx={{
                '& .MuiDialog-paper': {
                    minWidth: 'min(100%, 600px)',
                    minHeight: 'min(100%, 200px)',
                    position: 'absolute',
                    top: '5%',
                    overflowY: 'visible',
                }
            }}
        >
            <DialogTitle
                id={titleId}
                helpText={t(`CommandPaletteHelp`)}
                title={t(`CommandPaletteTitle`)}
                onClose={close}
            />
            <DialogContent sx={{
                background: palette.background.default,
                position: 'relative',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                overflowY: 'visible',
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
                    session={session}
                    showSecondaryLabel={true}
                    sxs={{
                        root: { width: '100%' },
                        paper: { background: palette.background.paper },
                    }}
                />
            </DialogContent>
        </Dialog >
    );
}