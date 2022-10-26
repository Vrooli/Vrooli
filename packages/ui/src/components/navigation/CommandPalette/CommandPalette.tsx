import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    Dialog,
    DialogContent,
    useTheme,
} from '@mui/material';
import { actionsItems, getObjectSlug, getObjectUrlBase, getUserLanguages, listToAutocomplete, PubSub, shortcutsItems } from 'utils';
import { AutocompleteSearchBar } from 'components/inputs';
import { APP_LINKS } from '@shared/consts';
import { AutocompleteOption } from 'types';
import { useLazyQuery } from '@apollo/client';
import { CommandPaletteProps } from '../types';
import { homePage, homePageVariables } from 'graphql/generated/homePage';
import { homePageQuery } from 'graphql/query';
import { useLocation } from '@shared/route';
import { DialogTitle } from 'components';
import { uuidValidate } from '@shared/uuid';

const helpText =
    `Use this dialog to quickly navigate to other pages.

It can be opened at any time by entering CTRL + P.`

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

const titleAria = 'command-palette-dialog-title';

export const CommandPalette = ({
    session
}: CommandPaletteProps) => {
    const { palette } = useTheme();
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

    const [refetch, { data, loading }] = useLazyQuery<homePage, homePageVariables>(homePageQuery, { variables: { input: { searchString: searchString.replaceAll(/![^\s]{1,}/g, '') } }, errorPolicy: 'all' });
    useEffect(() => { open && refetch() }, [open, refetch, searchString]);


    const autocompleteOptions: AutocompleteOption[] = useMemo(() => {
        const firstResults: AutocompleteOption[] = [];
        // If "help" typed, add help and faq shortcuts as first result
        if (searchString.toLowerCase().startsWith('help')) {
            firstResults.push({
                __typename: "Shortcut",
                label: `Help - Beginner's Guide`,
                id: APP_LINKS.Welcome,
            }, {
                __typename: "Shortcut",
                label: 'Help - FAQ',
                id: APP_LINKS.FAQ,
            });
        }
        // Group all query results and sort by number of stars
        const flattened = (Object.values(data?.homePage ?? [])).reduce((acc, curr) => acc.concat(curr), []);
        const queryItems = listToAutocomplete(flattened, languages).sort((a: any, b: any) => {
            return b.stars - a.stars;
        });
        return [...firstResults, ...queryItems, ...shortcutsItems, ...actionsItems];
    }, [languages, data, searchString]);

    /**
     * When an autocomplete item is selected, navigate to object
     */
    const onInputSelect = useCallback((newValue: AutocompleteOption) => {
        if (!newValue) return;
        // Clear search string and close command palette
        close();
        setSearchString('');
        // If selected item is an action (i.e. no navigation required), do nothing 
        // (search bar performs actions automatically)
        if (newValue.__typename === 'Action') {
            return;
        }
        let newLocation: string;
        // If selected item is a shortcut, newLocation is in the id field
        if (newValue.__typename === 'Shortcut') {
            newLocation = newValue.id
        }
        // Otherwise, object url must be constructed
        else {
            newLocation = `${getObjectUrlBase(newValue)}/${getObjectSlug(newValue)}`
        }
        // If new pathname is the same, reload page
        const shouldReload = stripUrl(`${window.location.origin}${newLocation}`) === stripUrl(window.location.href);
        // Set new location
        setLocation(newLocation);
        if (shouldReload) {
            window.location.reload();
        }
    }, [close, setLocation]);

    return (
        <Dialog
            open={open}
            onClose={close}
            aria-labelledby={titleAria}
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
                ariaLabel={titleAria}
                helpText={helpText}
                title={'What would you like to do?'}
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
                <AutocompleteSearchBar
                    id="command-palette-search"
                    autoFocus={true}
                    placeholder='Search content and quickly navigate the site'
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