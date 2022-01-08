import { Autocomplete, Box, IconButton, Input, Paper, Stack, Typography } from '@mui/material';
import { useState, useCallback, useEffect, useMemo } from 'react';
import { Search as SearchIcon } from '@mui/icons-material';
import { centeredDiv, centeredText } from 'styles';
import { autocomplete, autocompleteVariables } from 'graphql/generated/autocomplete';
import { useQuery } from '@apollo/client';
import { autocompleteQuery } from 'graphql/query';
import { debounce } from 'lodash';
import { ActorCard, FeedList, OrganizationCard, ProjectCard, RoutineCard, StandardCard } from 'components';

/**
 * Containers a search bar, lists of routines, projects, tags, and organizations, 
 * and a FAQ section.
 * If a search string is entered, each list is filtered by the search string. 
 * Otherwise, each list shows popular items. Each list has a "See more" button, 
 * which opens a full search page for that object type.
 */
export const HomePage = () => {
    const [searchString, setSearchString] = useState<string>('');
    const updateSearch = useCallback((e: any, newValue: any) => setSearchString(newValue), []);
    const { data, refetch } = useQuery<autocomplete, autocompleteVariables>(autocompleteQuery, { variables: { input: { searchString } } });
    const debouncedRefetch = useCallback(() => debounce(() => refetch(), 200), [refetch]);
    useEffect(() => { debouncedRefetch() }, [debouncedRefetch, searchString]);
    const autocompleteOptions = useMemo(() => data?.autocomplete ?? [], [data]);

    console.log('AUTOCOMPLETE DATA', data)

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
                                {...params}
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
                    <FeedList title="Routines" data={[]} cardFactory={(d: any) => <RoutineCard data={d}/>} />
                    <FeedList title="Projects" data={[]} cardFactory={(d: any) => <ProjectCard data={d}/>} />
                    <FeedList title="Organizations" data={[]} cardFactory={(d: any) => <OrganizationCard data={d}/>} />
                    <FeedList title="Users" data={[]} cardFactory={(d: any) => <ActorCard data={d}/>} />
                    <FeedList title="Standards" data={[]} cardFactory={(d: any) => <StandardCard data={d}/>} />
            </Stack>
        </Box>
    )
}