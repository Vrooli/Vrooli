import { useQuery } from "@apollo/client";
import { ORGANIZATION_SORT_BY } from "@local/shared";
import { Grid, Theme } from "@mui/material";
import { makeStyles } from "@mui/styles";
import { OrganizationCard, SearchBar, Selector } from "components";
import { OrganizationSortBy } from "graphql/generated/globalTypes";
import { organizations, organizationsVariables } from "graphql/generated/organizations";
import { organizationsQuery } from "graphql/query";
import { useCallback, useMemo, useState } from "react";
import { combineStyles } from "utils";
import { searchStyles } from ".";

const componentStyles = (theme: Theme) => ({
    
});

const useStyles = makeStyles(combineStyles(searchStyles, componentStyles));

const SORT_OPTIONS: {label: string, value: OrganizationSortBy}[] = Object.values(ORGANIZATION_SORT_BY).map((sortOption) => ({ label: sortOption, value: sortOption as OrganizationSortBy }));

export const SearchOrganizationsPage = () => {
    const classes = useStyles();
    const [searchString, setSearchString] = useState<string>('');
    const [sortBy, setSortBy] = useState<OrganizationSortBy>(SORT_OPTIONS[1].value);
    const { data } = useQuery<organizations, organizationsVariables>(organizationsQuery, { variables: { input: { sortBy, searchString } }, pollInterval: 50000 });

    console.log('SEARCH DATA', data);

    const cards = useMemo(() => (
        data?.organizations?.edges?.map((edge, index) =>
            <OrganizationCard
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