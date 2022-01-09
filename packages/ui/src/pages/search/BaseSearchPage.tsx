import { useQuery } from "@apollo/client";
import { Box, Button, FormControlLabel, Grid, List, Switch, Tooltip, Typography } from "@mui/material";
import { SearchBar, SortMenu, TimeMenu } from "components";
import { useCallback, useEffect, useMemo, useState } from "react";
import { centeredText, containerShadow, centeredDiv } from "styles";
import { BaseSearchPageProps, SearchQueryVariablesInput } from "./types";
import {
    AccessTime as TimeIcon,
    Sort as SortListIcon,
} from '@mui/icons-material';

export function BaseSearchPage<DataType, SortBy, Query, QueryVariables extends SearchQueryVariablesInput<SortBy>>({
    title = 'Search',
    sortOptions,
    defaultSortOption,
    query,
    listItemFactory,
}: BaseSearchPageProps<DataType, SortBy>) {
    const [searchString, setSearchString] = useState<string>('');
    const [sortAnchorEl, setSortAnchorEl] = useState(null);
    const [timeAnchorEl, setTimeAnchorEl] = useState(null);
    const [sortBy, setSortBy] = useState<SortBy | undefined>(defaultSortOption ?? sortOptions.length > 0 ? sortOptions[0].value : undefined);
    const [sortByLabel, setSortByLabel] = useState<string>(defaultSortOption ?? sortOptions.length > 0 ? sortOptions[0].label : undefined);
    const [createdTimeFrame, setCreatedTimeFrame] = useState<any | undefined>(undefined);
    const [createdTimeFrameLabel, setCreatedTimeFrameLabel] = useState<string>('Time');
    const { data, refetch } = useQuery<Query, QueryVariables>(query, { variables: ({ input: { sortBy, searchString, createdTimeFrame } } as any) });

    useEffect(() => { refetch() }, [refetch, searchString, sortBy]);

    const listItems = useMemo(() => data ? ((Object.values(data) as any)?.edges?.map((edge, index) => listItemFactory(edge.node, index))) : null, [listItemFactory, data]);

    const handleSort = useCallback((e) => setSortBy(e.target.value), []);
    const handleSearch = useCallback((newString) => setSearchString(newString), []);

    const handleSortOpen = (event) => setSortAnchorEl(event.currentTarget);
    const handleSortClose = (label?: string, selected?: string) => {
        setSortAnchorEl(null);
        if (selected) setSortBy(selected as any);
        if (label) setSortByLabel(label);
    };

    const handleTimeOpen = (event) => setTimeAnchorEl(event.currentTarget);
    const handleTimeClose = (label?: string, after?: Date | null, before?: Date | null) => {
        setTimeAnchorEl(null);
        if (!after && !before) setCreatedTimeFrame(undefined);
        else setCreatedTimeFrame({ after, before });
        if (label) setCreatedTimeFrameLabel(label);
    };

    return (
        <Box id="page">
            <SortMenu
                sortOptions={sortOptions}
                anchorEl={sortAnchorEl}
                onClose={handleSortClose}
            />
            <TimeMenu
                anchorEl={timeAnchorEl}
                onClose={handleTimeClose}
            />
            <Typography component="h2" variant="h4" sx={{ ...centeredText, paddingTop: 2 }}>{title}</Typography>
            <Grid container spacing={2}>
                <Grid item xs={12} sm={8}>
                    <SearchBar fullWidth value={searchString} onChange={handleSearch} />
                </Grid>
                <Grid item xs={6} sm={2}>
                    <Button
                        color="secondary"
                        fullWidth
                        startIcon={<SortListIcon />}
                        onClick={handleSortOpen}
                        sx={{
                            height: '100%',
                        }}
                    >
                        {sortByLabel}
                    </Button>
                </Grid>
                <Grid item xs={6} sm={2}>
                    <Button
                        color="secondary"
                        fullWidth
                        startIcon={<TimeIcon />}
                        onClick={handleTimeOpen}
                        sx={{
                            height: '100%',
                        }}
                    >
                        {createdTimeFrameLabel}
                    </Button>
                </Grid>
            </Grid>
            <Box
                sx={{
                    ...containerShadow,
                    borderRadius: '8px',
                    marginTop: 2,
                    background: (t) => t.palette.background.default,
                }}
            >
                <List>
                    {listItems}
                </List>
            </Box>
            {/* <Box sx={{ ...centeredDiv, paddingTop: 1 }}>
                <Typography component="h2" variant="h4" sx={{ ...centeredText }}>
                    Couldn't find what you were looking for? Try creating your own!
                </Typography>
                <Button>
                    New 
                </Button>
            </Box> */}
        </Box>
    )
}