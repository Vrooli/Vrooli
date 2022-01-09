import { Autocomplete, Box, IconButton, Input, Paper, Stack, TextField, Typography } from '@mui/material';
import { useState, useCallback, useEffect, useMemo } from 'react';
import { Search as SearchIcon } from '@mui/icons-material';
import { centeredDiv, centeredText } from 'styles';
import { autocomplete, autocompleteVariables } from 'graphql/generated/autocomplete';
import { useQuery } from '@apollo/client';
import { autocompleteQuery } from 'graphql/query';
import { debounce } from 'lodash';
import { ActorCard, ActorListItem, FeedList, OrganizationCard, ProjectCard, RoutineCard, StandardCard } from 'components';
import { useLocation } from 'wouter';
import { APP_LINKS } from '@local/shared';

/**
 * Containers a search bar, lists of routines, projects, tags, and organizations, 
 * and a FAQ section.
 * If a search string is entered, each list is filtered by the search string. 
 * Otherwise, each list shows popular items. Each list has a "See more" button, 
 * which opens a full search page for that object type.
 */
export const HomePage = () => {
    const [, setLocation] = useLocation();
    const [searchString, setSearchString] = useState<string>('');
    const updateSearch = useCallback((e: any, newValue: any) => setSearchString(newValue), []);
    const { data, refetch } = useQuery<autocomplete, autocompleteVariables>(autocompleteQuery, { variables: { input: { searchString } } });
    const debouncedRefetch = useCallback(() => debounce(() => refetch(), 200), [refetch]); //TODO debounce not working
    useEffect(() => { debouncedRefetch() }, [debouncedRefetch, searchString]);
    const { routines, projects, organizations, standards, users } = useMemo(() => {
        if (!data) return { routines: [], projects: [], organizations: [], standards: [], users: [] };
        const { routines, projects, organizations, standards, users } = data.autocomplete;
        return { routines, projects, organizations, standards, users };
    }, [data]);
    const autocompleteOptions = useMemo(() => {
        if (!data) return [];
        const routines = data.autocomplete.routines.map(r => ({ title: r.title, id: r.id, stars: r.stars, objectType: 'Routine' }));
        const projects = data.autocomplete.projects.map(p => ({ title: p.name, id: p.id, stars: p.stars, objectType: 'Project' }));
        const organizations = data.autocomplete.organizations.map(o => ({ title: o.name, id: o.id, stars: o.stars, objectType: 'Organization' }));
        const standards = data.autocomplete.standards.map(s => ({ title: s.name, id: s.id, stars: s.stars, objectType: 'Standard' }));
        const users = data.autocomplete.users.map(u => ({ title: u.username, id: u.id, stars: u.stars, objectType: 'User' }));
        return [...routines, ...projects, ...organizations, ...standards, ...users].sort((a: any, b: any) => {
            return b.stars - a.stars;
        });
    }, [data]);

    console.log('AUTOCOMPLETE', autocompleteOptions)

    // Feed title is Popular when no search
    const getFeedTitle = useCallback((objectName: string) => {
        const prefix = Boolean(searchString) ? 'Popular ' : '';
        return `${prefix}${objectName}`;
    }, [searchString]);

    // Opens correct search page
    const openSearch = useCallback((linkBase: string, id?: string) => {
        return id ? `${linkBase}/${id}` : linkBase;
    }, [])

    return (
        <Box id="page">
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
            <Stack spacing={2} direction="column" sx={{ ...centeredDiv, paddingTop: '40px', paddingBottom: '40px' }}>
                <Typography component="h2" variant="h5">Examples</Typography>
            </Stack>
            {/* Result feeds (or popular feeds if no search string) */}
            <Stack spacing={10} direction="column">
                <FeedList
                    title={getFeedTitle('Users')}
                    onClick={(id?: string) => openSearch(APP_LINKS.Profile, id)}
                >
                    {users.map(o => (
                        <ActorListItem key={`user-list-${o.id}`} data={o} isStarred={false} isOwn={false} onClick={() => {}} onStarClick={() => {}} />
                    ))}
                </FeedList>
                {/* <FeedList 
                    title={getFeedTitle(AutocompleteResultType.Routine)}
                    data={autocompleteOptions.filter(o => o.objectType === AutocompleteResultType.Routine)}
                    createListItem={(d: any) => <ActorListItem data={d} isStarred={false} isOwn={false} onClick={() => {}} onStarClick={() => {}} />} 
                    onClick={(id?: string) => openSearch(AutocompleteResultType.Routine, id)}
                /> */}
                {/* <FeedList 
                    title={getFeedTitle(AutocompleteResultType.Project)}
                    data={autocompleteOptions.filter(o => o.objectType === AutocompleteResultType.Project)}
                    cardFactory={(d: any) => <ProjectCard data={d} />} 
                    onClick={(id?: string) => openSearch(AutocompleteResultType.Project, id)}
                />
                <FeedList 
                    title={getFeedTitle(AutocompleteResultType.Organization)}
                    data={autocompleteOptions.filter(o => o.objectType === AutocompleteResultType.Organization)}
                    cardFactory={(d: any) => <OrganizationCard data={d} />} 
                    onClick={(id?: string) => openSearch(AutocompleteResultType.Organization, id)}
                />
                <FeedList 
                    title={getFeedTitle(AutocompleteResultType.User)}
                    data={autocompleteOptions.filter(o => o.objectType === AutocompleteResultType.User)}
                    cardFactory={(d: any) => <ActorCard data={d} />} 
                    onClick={(id?: string) => openSearch(AutocompleteResultType.User, id)}
                />
                <FeedList 
                    title={getFeedTitle(AutocompleteResultType.Standard)}
                    data={autocompleteOptions.filter(o => o.objectType === AutocompleteResultType.Standard)}
                    cardFactory={(d: any) => <StandardCard data={d} />} 
                    onClick={(id?: string) => openSearch(AutocompleteResultType.Standard, id)}
                /> */}
            </Stack>
        </Box>
    )
}