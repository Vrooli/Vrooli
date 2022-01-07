import { makeStyles } from '@mui/styles';
import { IconButton, InputBase, Paper, Stack, Theme, Typography } from '@mui/material';
import { useState, useCallback } from 'react';
import { Search as SearchIcon } from '@mui/icons-material';

const useStyles = makeStyles((theme: Theme) => ({
    root: {},
    title: {
        textAlign: 'center',
    },
}));

/**
 * Containers a search bar, lists of routines, projects, tags, and organizations, 
 * and a FAQ section.
 * If a search string is entered, each list is filtered by the search string. 
 * Otherwise, each list shows popular items. Each list has a "See more" button, 
 * which opens a full search page for that object type.
 */
export const HomePage = ({

}) => {
    const classes = useStyles();
    const [searchString, setSearchString] = useState<string>('');
    const handleSearch = useCallback((e) => setSearchString(e.target.value), []);

    return (
        <div id="page" className={classes.root}>
            <Stack spacing={2} direction="column" sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center'}}>
                <Typography component="h1" variant="h2" className={classes.title}>What would you like to do?</Typography>
                {/* ========= #region Custom SearchBar ========= */}
                <Paper
                    component="form"
                    sx={{ p: '2px 4px', display: 'flex', alignItems: 'center', width: 'min(100%, 600px)', borderRadius: '10px' }}
                >
                    <InputBase
                        sx={{ ml: 1, flex: 1 }}
                        value={searchString}
                        onChange={handleSearch}
                        placeholder='Search routines, projects, and more...'
                        inputProps={{ 'aria-label': 'main-search-textfield' }}
                    />
                    <IconButton sx={{ p: '10px' }} aria-label="search">
                        <SearchIcon />
                    </IconButton>
                </Paper>
                {/* =========  #endregion ========= */}
            </Stack>
            <Typography component="h2" variant="h4">Examples</Typography>
        </div>
    )
}