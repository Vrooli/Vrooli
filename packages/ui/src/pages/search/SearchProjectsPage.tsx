import { ProjectSortBy } from "@local/shared";
import { ProjectListItem } from "components";
import { projectsQuery } from "graphql/query";
import { Project } from "types";
import { BaseSearchPage } from "./BaseSearchPage";

const SORT_OPTIONS: {label: string, value: ProjectSortBy}[] = Object.values(ProjectSortBy).map((sortOption) => ({ label: sortOption, value: sortOption as ProjectSortBy }));

export const SearchProjectsPage = () => {
    const listItemFactory = (node: Project, index: number) => (
        <ProjectListItem 
            key={`project-list-item-${index}`} 
            data={node} 
            isStarred={false}
            isOwn={false}
            onClick={() => {}}
            onStarClick={() => {}}
        />)

    return (
        <BaseSearchPage 
            title={'Search Projects'}
            sortOptions={SORT_OPTIONS}
            defaultSortOption={SORT_OPTIONS[1].value}
            query={projectsQuery}
            listItemFactory={listItemFactory}
        />
    )
}