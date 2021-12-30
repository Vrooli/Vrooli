import { useQuery } from "@apollo/client";
import { ProjectSortBy } from "@local/shared";
import { Grid, Theme } from "@mui/material";
import { makeStyles } from "@mui/styles";
import { ProjectCard, SearchBar, Selector } from "components";
import { ProjectSortBy } from "graphql/generated/globalTypes";
import { projects, projectsVariables } from "graphql/generated/projects";
import { projectsQuery } from "graphql/query";
import { useCallback, useMemo, useState } from "react";
import { combineStyles } from "utils";
import { searchStyles } from ".";

const componentStyles = (theme: Theme) => ({
    
});

const useStyles = makeStyles(combineStyles(searchStyles, componentStyles));

const SORT_OPTIONS: {label: string, value: ProjectSortBy}[] = Object.values(ProjectSortBy).map((sortOption) => ({ label: sortOption, value: sortOption as ProjectSortBy }));

export const SearchProjectsPage = () => {
    const classes = useStyles();
    const [searchString, setSearchString] = useState<string>('');
    const [sortBy, setSortBy] = useState<ProjectSortBy>(SORT_OPTIONS[1].value);
    const { data } = useQuery<projects, projectsVariables>(projectsQuery, { variables: { input: { sortBy, searchString } }, pollInterval: 50000 });

    console.log('SEARCH DATA', data);

    const cards = useMemo(() => (
        data?.projects?.edges?.map((edge, index) =>
            <ProjectCard
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