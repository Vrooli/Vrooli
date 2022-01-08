import { Autocomplete, Box, IconButton, Input, Paper, Stack, TextField, Typography } from '@mui/material';
import { useState, useCallback, useEffect, useMemo } from 'react';
import { Search as SearchIcon } from '@mui/icons-material';
import { centeredDiv, centeredText } from 'styles';
import { autocomplete, autocompleteVariables } from 'graphql/generated/autocomplete';
import { useQuery } from '@apollo/client';
import { autocompleteQuery } from 'graphql/query';
import { debounce } from 'lodash';
import { ActorCard, FeedList, OrganizationCard, ProjectCard, RoutineCard, StandardCard } from 'components';
import { Navigate, useNavigate } from 'react-router-dom';
import { APP_LINKS, AutocompleteResultType } from '@local/shared';

/**
 * Containers a search bar, lists of routines, projects, tags, and organizations, 
 * and a FAQ section.
 * If a search string is entered, each list is filtered by the search string. 
 * Otherwise, each list shows popular items. Each list has a "See more" button, 
 * which opens a full search page for that object type.
 */
export const HomePage = () => {
    const navigate = useNavigate();
    const [searchString, setSearchString] = useState<string>('');
    const updateSearch = useCallback((e: any, newValue: any) => setSearchString(newValue), []);
    const { data, refetch } = useQuery<autocomplete, autocompleteVariables>(autocompleteQuery, { variables: { input: { searchString } } });
    const debouncedRefetch = useCallback(() => debounce(() => refetch(), 200), [refetch]); //TODO debounce not working
    useEffect(() => { debouncedRefetch() }, [debouncedRefetch, searchString]);
    const autocompleteOptions = useMemo(() => data?.autocomplete ?? [], [data]);

    console.log('AUTOCOMPLETE', autocompleteOptions)

    // Feed title is Popular when no search
    const getFeedTitle = useCallback((objectType: AutocompleteResultType) => {
        const prefix = Boolean(searchString) ? 'Popular ' : '';
        switch (objectType) {
            case AutocompleteResultType.Organization:
                return prefix + 'Organizations';
            case AutocompleteResultType.Project:
                return prefix + 'Projects';
            case AutocompleteResultType.Routine:
                return prefix + 'Routines';
            case AutocompleteResultType.Standard:
                return prefix + 'Standards';
            default AutocompleteResultType.User:
                return prefix + 'Users';
        }
    }, [searchString]);

    // Opens correct search page
    const openSearch = useCallback((objectType: AutocompleteResultType, id?: string) => {
        const navHelper = (baseLink: string) => navigate(id ? `${baseLink}/${id}` : baseLink);
        switch (objectType) {
            case AutocompleteResultType.Organization:
                return navHelper(APP_LINKS.SearchOrganizations);
            case AutocompleteResultType.Project:
                return navHelper(APP_LINKS.SearchProjects);
            case AutocompleteResultType.Routine:
                return navHelper(APP_LINKS.SearchRoutines);
            case AutocompleteResultType.Standard:
                return navHelper(APP_LINKS.SearchStandards);
            default AutocompleteResultType.User:
                return navHelper(APP_LINKS.SearchUsers);
        }
    }, [])

    return (
        <Box
            id="page"
            sx={{
                background: 'fixed radial-gradient(circle, rgba(208,213,226,1) 7%, rgba(179,191,217,1) 66%, rgba(160,188,249,1) 94%)',
            }}
        >
            {/* Prompt stack */}
            <Stack spacing={2} direction="column" sx={{ ...centeredDiv, paddingTop: { xs: '5vh', sm: '30vh' } }}>
                <Typography component="h1" variant="h2" sx={{ ...centeredText }}>What would you like to do?</Typography>
                {/* ========= #region Custom SearchBar ========= */}
                <Autocomplete
                    disablePortal
                    id="main-search"
                    options={autocompleteOptions}
                    getOptionLabel={(o: any) => `${o.title} | ${o.objectType}`}
                    sx={{ width: 'min(100%, 600px)' }}
                    onChange={updateSearch}
                    onInputChange={updateSearch}
                    renderInput={(params) => (
                        < Paper
                            component="form"
                            sx={{ p: '2px 4px', display: 'flex', alignItems: 'center', borderRadius: '10px' }}
                        >
                            <Input
                                sx={{ ml: 1, flex: 1 }}
                                disableUnderline={true}
                                value={searchString}
                                placeholder='Search routines, projects, and more...'
                                ref={params.InputProps.ref}
                                inputProps={params.inputProps}
                                autoFocus
                            />
                            <IconButton sx={{ p: '10px' }} aria-label="main-search-icon">
                                <SearchIcon />
                            </IconButton>
                        </Paper>
                    )
                    }
                />
                {/* =========  #endregion ========= */}
            </Stack>
            {/* Examples stack */}
            <Stack spacing={2} direction="column" sx={{ ...centeredDiv, paddingTop: '40px' }}>
                <Typography component="h2" variant="h5">Examples</Typography>
            </Stack>
            {/* Result feeds (or popular feeds if no search string) */}
            <Stack spacing={10} direction="column">
                <FeedList 
                    title={getFeedTitle("Routines")}
                    data={[]} 
                    cardFactory={(d: any) => <RoutineCard data={d} />} 
                    onClick={() => {}}
                />
                <FeedList 
                    title={getFeedTitle("Projects")}
                    data={[]} 
                    cardFactory={(d: any) => <ProjectCard data={d} />} 
                    onClick={() => {}}
                />
                <FeedList 
                    title={getFeedTitle("Organizations")}
                    data={[]} 
                    cardFactory={(d: any) => <OrganizationCard data={d} />} 
                    onClick={() => {}}
                />
                <FeedList 
                    title={getFeedTitle("Users")}
                    data={[]} 
                    cardFactory={(d: any) => <ActorCard data={d} />} 
                    onClick={() => {}}
                />
                <FeedList 
                    title={getFeedTitle("Standards")}
                    data={[]} 
                    cardFactory={(d: any) => <StandardCard data={d} />} 
                    onClick={() => {}}
                />
            </Stack>
        </Box>
    )
}