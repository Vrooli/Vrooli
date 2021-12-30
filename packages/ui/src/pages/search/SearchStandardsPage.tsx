import { useQuery } from "@apollo/client";
import { StandardSortBy } from "@local/shared";
import { Grid, Theme } from "@mui/material";
import { makeStyles } from "@mui/styles";
import { StandardCard, SearchBar, Selector } from "components";
import { StandardSortBy } from "graphql/generated/globalTypes";
import { standards, standardsVariables } from "graphql/generated/standards";
import { standardsQuery } from "graphql/query";
import { useCallback, useMemo, useState } from "react";
import { combineStyles } from "utils";
import { searchStyles } from ".";

const componentStyles = (theme: Theme) => ({
    
});

const useStyles = makeStyles(combineStyles(searchStyles, componentStyles));

const SORT_OPTIONS: {label: string, value: StandardSortBy}[] = Object.values(StandardSortBy).map((sortOption) => ({ label: sortOption, value: sortOption as StandardSortBy }));

export const SearchStandardsPage = () => {
    const classes = useStyles();
    const [searchString, setSearchString] = useState<string>('');
    const [sortBy, setSortBy] = useState<StandardSortBy>(SORT_OPTIONS[1].value);
    const { data } = useQuery<standards, standardsVariables>(standardsQuery, { variables: { input: { sortBy, searchString } }, pollInterval: 50000 });

    console.log('SEARCH DATA', data);

    const cards = useMemo(() => (
        data?.standards?.edges?.map((edge, index) =>
            <StandardCard
                key={index}
                data={edge.node}
            />)
    ), [data])

    const handleSort = useCallback((e) => setSortBy(e.target.value), []);
    const handleSearch = useCallback((newString) => setSearchString(newString), []);

    return (
        <div id="page">
            <Grid container spacing={2}>
                <Grid item xs={12} sm={6}>
                    <Selector
                        className={classes.selector}
                        fullWidth
                        options={SORT_OPTIONS}
                        selected={sortBy}
                        handleChange={handleSort}
                        inputAriaLabel='sort-actors-selector-label'
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