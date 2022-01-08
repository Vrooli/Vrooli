import { useQuery } from "@apollo/client";
import { Grid, Theme, Typography } from "@mui/material";
import { makeStyles } from "@mui/styles";
import { SearchBar, Selector } from "components";
import { useCallback, useEffect, useMemo, useState } from "react";
import { BaseSearchPageProps, SearchQueryVariablesInput } from "./types";

const useStyles = makeStyles((theme: Theme) => ({
    title: {
        textAlign: 'center',
    },
    cardFlex: {
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
        alignItems: 'stretch',
    },
    selector: {
        marginBottom: '1em',
    },
}));

export function BaseSearchPage<DataType, SortBy, Query, QueryVariables extends SearchQueryVariablesInput<SortBy>>({
    title = 'Search',
    sortOptions,
    defaultSortOption,
    query,
    cardFactory,
}: BaseSearchPageProps<DataType, SortBy>) {
    const classes = useStyles();
    const [searchString, setSearchString] = useState<string>('');
    const [sortBy, setSortBy] = useState<SortBy | undefined>(defaultSortOption ?? sortOptions.length > 0 ? sortOptions[0].value : undefined);
    const { data, refetch } = useQuery<Query, QueryVariables>(query, { variables: ({ input: { sortBy, searchString } } as any) });

    useEffect(() => { refetch() }, [refetch, searchString, sortBy]);

    const cards = useMemo(() => data ? ((Object.values(data) as any)?.edges?.map((edge, index) => cardFactory(edge.node, index))) : null, [cardFactory, data]);

    const handleSort = useCallback((e) => setSortBy(e.target.value), []);
    const handleSearch = useCallback((newString) => setSearchString(newString), []);

    return (
        <div id="page">
            <Typography component="h2" variant="h4" className={classes.title}>{title}</Typography>
            <Grid container spacing={2}>
                <Grid item xs={12} sm={6}>
                    <Selector
                        className={classes.selector}
                        fullWidth
                        options={sortOptions}
                        selected={sortBy}
                        handleChange={handleSort}
                        inputAriaLabel='sort-selector-label'
                        label="Sort" />
                </Grid>
                <Grid item xs={12} sm={6}>
                    <SearchBar fullWidth value={searchString} onChange={handleSearch} />
                </Grid>
            </Grid>
            <div className={classes.cardFlex}>
                {cards}
            </div>
        </div>
    )
}