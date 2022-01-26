import { APP_LINKS } from "@local/shared";
import { projectDefaultSortOption, ProjectListItem, projectOptionLabel, ProjectSortOptions, ProjectView, ShareDialog, ViewDialogBase } from "components";
import { projectsQuery } from "graphql/query";
import { useCallback, useMemo, useState } from "react";
import { Project } from "types";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchProjectsPageProps } from "./types";
import { useLocation, useRoute } from "wouter";

export const SearchProjectsPage = ({
    session
}: SearchProjectsPageProps) => {
    const [, setLocation] = useLocation();
    const [match, params] = useRoute(`${APP_LINKS.SearchProjects}/:id`);
    // Handles dialog when selecting a search result
    const [selected, setSelected] = useState<Project | undefined>(undefined);
    const selectedDialogOpen = Boolean(match || selected);
    const handleSelected = useCallback((selected: Project) => {
        setSelected(selected);
        setLocation(`${APP_LINKS.SearchProjects}/${selected.id}`);
    }, [setLocation]);
    const handleSelectedDialogClose = useCallback(() => {
        setSelected(undefined);
        // If selected data exists, then we know we can go back to the previous page
        if (selected) window.history.back();
        // Otherwise the user must have entered the page directly, so we can navigate to the search page
        else setLocation(APP_LINKS.SearchProjects);
    }, [setLocation, selected]);

    const partialData = useMemo(() => {
        if (selected) return selected;
        if (params?.id) return { id: params.id };
        return undefined;
    }, [params, selected]);

    // Handles dialog for the button that appears after scrolling a certain distance
    const [surpriseDialogOpen, setSurpriseDialogOpen] = useState(false);
    const handleSurpriseDialogOpen = useCallback(() => setSurpriseDialogOpen(true), []);
    const handleSurpriseDialogClose = useCallback(() => setSurpriseDialogOpen(false), []);

    const listItemFactory = (node: Project, index: number) => (
        <ProjectListItem
            key={`project-list-item-${index}`}
            session={session}
            data={node}
            isOwn={false}
            onClick={(selected: Project) => setSelected(selected)}
        />)

    return (
        <>
            {/* Invite link dialog */}
            <ShareDialog onClose={handleSurpriseDialogClose} open={surpriseDialogOpen} />
            {/* Selected dialog */}
            <ViewDialogBase
                title='View Project'
                open={selectedDialogOpen}
                onClose={handleSelectedDialogClose}
            >
                <ProjectView session={session} partialData={partialData} />
            </ViewDialogBase>
            {/* Search component */}
            <BaseSearchPage
                title="Projects"
                searchPlaceholder="Search..."
                sortOptions={ProjectSortOptions}
                defaultSortOption={projectDefaultSortOption}
                query={projectsQuery}
                listItemFactory={listItemFactory}
                getOptionLabel={(o: any) => o.name}
                onObjectSelect={projectOptionLabel}
                showAddButton={true}
                onAddClick={() => {}}
                popupButtonText="Add"
                popupButtonTooltip="Can't find wha you're looking for? Create it!😎"
                onPopupButtonClick={handleSurpriseDialogOpen}
            />
        </>
    )
}