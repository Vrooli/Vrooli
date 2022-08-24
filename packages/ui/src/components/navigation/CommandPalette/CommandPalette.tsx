import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    Dialog,
    DialogContent,
    DialogTitle,
    IconButton,
    Typography,
    useTheme,
} from '@mui/material';
import {
    Close as CloseIcon,
} from '@mui/icons-material';
import { noSelect } from 'styles';
import { listToAutocomplete, openObject, PubSub, shortcutsItems } from 'utils';
import { AutocompleteSearchBar } from 'components/inputs';
import { APP_LINKS } from '@shared/consts';
import { AutocompleteOption } from 'types';
import { useQuery } from '@apollo/client';
import { CommandPaletteProps } from '../types';
import { homePage, homePageVariables } from 'graphql/generated/homePage';
import { homePageQuery } from 'graphql/query';
import { useLocation } from '@shared/route';

const CommandPalette = ({
    session
}: CommandPaletteProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const languages = useMemo(() => session?.languages ?? navigator.languages, [session]);

    const [open, setOpen] = useState(false);
    const close = useCallback(() => setOpen(false), []);

    useEffect(() => {
        let dialogSub = PubSub.get().subscribeCommandPalette(() => setOpen(o => !o));
        return () => { PubSub.get().unsubscribe(dialogSub) };
    }, [])

    const [searchString, setSearchString] = useState<string>('');
    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue) }, []);

    const { data, refetch, loading } = useQuery<homePage, homePageVariables>(homePageQuery, { variables: { input: { searchString: searchString.replaceAll(/![^\s]{1,}/g, '') } }, errorPolicy: 'all' });
    useEffect(() => { refetch() }, [refetch, searchString]);


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
        return [...firstResults, ...queryItems, ...shortcutsItems];
    }, [languages, data, searchString]);

    /**
     * When an autocomplete item is selected, navigate to object
     */
    const onInputSelect = useCallback((newValue: AutocompleteOption) => {
        if (!newValue) return;
        // Clear search string and close command palette
        setSearchString('');
        close();
        // Replace current state with search string, so that search is not lost. 
        // Only do this if the selected item is not a shortcut
        if (newValue.__typename !== 'Shortcut' && searchString) setLocation(`${APP_LINKS.Home}?search="${searchString}"`, { replace: true });
        else setLocation(APP_LINKS.Home, { replace: true });
        // If selected item is a shortcut, navigate to it
        if (newValue.__typename === 'Shortcut') {
            setLocation(newValue.id);
        }
        // Otherwise, navigate to item page
        else {
            openObject(newValue, setLocation);
        }
    }, [close, searchString, setLocation]);

    return (
        <Dialog
            open={open}
            onClose={close}
            aria-labelledby="command-palette-dialog-title"
            aria-describedby="command-palette-dialog-description"
            sx={{
                '& .MuiDialog-paper': {
                    border: palette.mode === 'dark' ? `1px solid white` : 'unset',
                    minWidth: 'min(100%, 600px)',
                    minHeight: 'min(100%, 200px)',
                    position: 'absolute',
                    top: '5%',
                    overflowY: 'visible',
                }
            }}
        >
            {/* Title with close icon */}
            <DialogTitle
                id="command-palette-dialog-title"
                sx={{
                    ...noSelect,
                    display: 'flex',
                    alignItems: 'center',
                    padding: 2,
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                }}
            >
                <Typography
                    variant="h6"
                    sx={{
                        width: '-webkit-fill-available',
                        textAlign: 'center',
                    }}
                >
                    What would you like to do?
                </Typography>
                <IconButton
                    aria-label="close"
                    edge="end"
                    onClick={close}
                >
                    <CloseIcon sx={{ fill: palette.primary.contrastText }} />
                </IconButton>
            </DialogTitle>
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

export { CommandPalette };