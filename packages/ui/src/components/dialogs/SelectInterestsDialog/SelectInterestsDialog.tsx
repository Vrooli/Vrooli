import { Box, Button, Dialog, Grid, IconButton, Stack, Typography } from '@mui/material';
import { SelectInterestsDialogProps } from '../types';
import {
    Cancel as CancelIcon,
    Close as CloseIcon,
    Save as SaveIcon,
} from '@mui/icons-material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import helpMarkdown from './interestsHelp.md';
import { HelpButton, Selector } from 'components';
import { tags, tagsVariables } from 'graphql/generated/tags';
import { tagsQuery } from 'graphql/query';
import { useQuery } from '@apollo/client';
import { Tag } from 'types';
import { TagSortBy } from '@local/shared';

export const SelectInterestsDialog = ({
    session,
    open,
    onClose,
    showHidden = false,
}: SelectInterestsDialogProps) => {
    // Store tag lists
    const [favorites, setFavorites] = useState<Tag[]>([]);
    const [hidden, setHidden] = useState<Tag[]>([]);
    const [popular, setPopular] = useState<Tag[]>([]);

    // Search for user's current tags, and popular tags
    const { data: currentFavorites, refetch: refetchCurrentFavorites } = useQuery<tags, tagsVariables>(tagsQuery, { variables: { input: { 
        myTags: true,
        sortBy: TagSortBy.AlphabeticalDesc,
        take: 100,
    } } });
    useEffect(() => { setFavorites(currentFavorites?.tags?.edges?.map(e => e.node) ?? []); }, [currentFavorites]);
    const { data: currentHidden, refetch: refetchCurrentHidden } = useQuery<tags, tagsVariables>(tagsQuery, { variables: { input: { 
        hidden: true,
        sortBy: TagSortBy.AlphabeticalDesc,
        take: 100,
    } } });
    useEffect(() => { setHidden(currentHidden?.tags?.edges?.map(e => e.node) ?? []); }, [currentHidden]);
    const { data: popularTags, refetch: refetchPopular } = useQuery<tags, tagsVariables>(tagsQuery, { variables: { input: { 
        sortBy: TagSortBy.AlphabeticalDesc,
        take: 100,
    } } });
    useEffect(() => { setPopular(popularTags?.tags?.edges?.map(e => e.node) ?? []); }, [popularTags]);

    // Autocomplete for favorite tags
    const [favoritesSearchString, setFavoritesSearchString] = useState<string>('');
    const updateFavoritesSearchString = useCallback((change: any) => { 
        console.log('updateFavoritesSearchString', change);
        // setFavoritesSearchString(newValue) 
    }, []);
    const { data: favoritesAutocompleteData, refetch: refetchFavoritesAutocomplete } = useQuery<tags, tagsVariables>(tagsQuery, { variables: { input: { 
        searchString: favoritesSearchString,
        sortBy: TagSortBy.AlphabeticalDesc,
        take: 25,
    } } });
    useEffect(() => { refetchFavoritesAutocomplete() }, [favoritesSearchString]);
    const favoritesAutocompleteOptions = useMemo(() => {
        if (!favoritesAutocompleteData) return [];
        return favoritesAutocompleteData.tags.edges.map(({ node }) => node);
    }, [favoritesAutocompleteData]);

    // Autocomplete for hidden tags
    const [hiddenSearchString, setHiddenSearchString] = useState<string>('');
    const updateHiddenSearchString = useCallback((newValue: any) => { setHiddenSearchString(newValue) }, []);
    const { data: hiddenAutocompleteData, refetch: refetchHiddenAutocomplete } = useQuery<tags, tagsVariables>(tagsQuery, { variables: { input: { 
        searchString: hiddenSearchString,
        sortBy: TagSortBy.AlphabeticalDesc,
        take: 25,
    } } });
    useEffect(() => { refetchHiddenAutocomplete() }, [hiddenSearchString]);
    const hiddenAutocompleteOptions = useMemo(() => {
        if (!hiddenAutocompleteData) return [];
        return hiddenAutocompleteData.tags.edges.map(({ node }) => node);
    }, [hiddenAutocompleteData]);

    // const onFavoriteOptionSelect = useCallback((_e: any, newValue: any) => {
    //     if (!newValue) return;
    //     // Determine object from selected label
    //     const selectedItem = favoritesAutocompleteOptions.find(o => `${o.tag} | ${o.description}` === newValue);
    //     if (!selectedItem) return;
    //     console.log('selectedItem', selectedItem);
    //     return openSearch(linkMap[selectedItem.objectType], selectedItem.id);
    // }, [autocompleteOptions]);

    // Return updated information
    const handleClose = useCallback(() => {
        // TODO if unsaved changes, prompt
        onClose();
    }, [onClose]);

    // Stores help text
    const [helpText, setHelpText] = useState<string>('');
    // Parse help text from markdown
    useEffect(() => {
        fetch(helpMarkdown)
            .then((response) => response.text())
            .then((text) => {
                setHelpText(text);
            });
    }, []);

    return (
        <Dialog
            onClose={handleClose}
            open={open}
            sx={{
                zIndex: 10000,
                '& .MuiDialogContent-root': {
                    overflow: 'hidden',
                    borderRadius: 2,
                    boxShadow: "0 0 35px 0 rgba(0,0,0,0.5)",
                    textAlign: "center",
                    padding: "1em",
                },
            }}
        >
            <Box sx={{
                padding: 2,
                background: "#072781",
                color: 'white',
                transition: 'background 0.2s ease-in-out',
            }}>
                <IconButton edge="end" color="inherit" onClick={handleClose} aria-label="close">
                    <CloseIcon
                        sx={{ fill: 'white' }}
                    />
                </IconButton>
                <Stack direction="column" spacing={1} mb={2} sx={{ alignItems: 'center' }}>
                    <Stack direction="row" spacing={2}>
                        <Typography variant="h4" component="h1">Select Your Interests</Typography>
                        <HelpButton markdown={helpText} />
                    </Stack>
                    <Typography variant="h6" mb={3}>These can be changed at any time</Typography>
                    {/* Favorites TODO not sure how to pass in/handle selector data. Might also need Autocomplete instead of selector */}
                    <Selector
                        id="favorite-tags-input"
                        placeholder='Enter topics you are interested in, followed by spaces or arrows...'
                        multiple
                        fullWidth
                        options={favoritesAutocompleteOptions}
                        getOptionLabel={(o: any) => `${o.tag} | ${o.description}`}
                        selected={favoritesSearchString}
                        handleChange={updateFavoritesSearchString}
                        // onInputChange={() => {}}
                        sx={{ width: 'min(100%, 600px)' }}
                    />
                    {/* Hidden */}
                    {/* Popular */}
                    {/* Save/Cancel buttons */}
                    <Grid item xs={12} sm={6}>
                        <Button
                            fullWidth
                            startIcon={<SaveIcon />}
                            onClick={() => { }}
                        >Save</Button>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                        <Button
                            fullWidth
                            startIcon={<CancelIcon />}
                            onClick={handleClose}
                        >Cancel</Button>
                    </Grid>
                </Stack>
            </Box>
        </Dialog>
    )
}