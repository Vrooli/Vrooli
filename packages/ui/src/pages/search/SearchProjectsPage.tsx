import { ProjectSortBy } from "@local/shared";
import { ProjectListItem } from "components";
import { projectsQuery } from "graphql/query";
import { useCallback, useState } from "react";
import { Project } from "types";
import { SortValueToLabelMap } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchSortBy } from "./types";

const SORT_OPTIONS: SearchSortBy<ProjectSortBy>[] = Object.values(ProjectSortBy).map((sortOption) => ({ 
    label: SortValueToLabelMap[sortOption], 
    value: sortOption as ProjectSortBy 
}));

export const SearchProjectsPage = () => {
    const [selected, setSelected] = useState<Project | undefined>(undefined);
    const dialogOpen = Boolean(selected);

    const handleDialogClose = useCallback(() => setSelected(undefined), []);

    const listItemFactory = (node: any, index: number) => (
        <ProjectListItem 
            key={`project-list-item-${index}`} 
            data={node} 
            isStarred={false}
            isOwn={false}
            onClick={(selected: Project) => setSelected(selected)}
            onStarClick={() => {}}
        />)

    return (
        <BaseSearchPage 
            title={'Search Projects'}
            searchPlaceholder="Search by name, description, or tags..."
            sortOptions={SORT_OPTIONS}
            defaultSortOption={SORT_OPTIONS[1]}
            query={projectsQuery}
            listItemFactory={listItemFactory}
            getOptionLabel={(o: any) => o.name}
            onObjectSelect={(selected: Project) => setSelected(selected)}
        />
    )
}