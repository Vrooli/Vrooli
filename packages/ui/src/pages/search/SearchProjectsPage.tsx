import { APP_LINKS, ProjectSortBy } from "@local/shared";
import { ProjectListItem, ProjectView, ShareDialog, ViewDialogBase } from "components";
import { projectsQuery } from "graphql/query";
import { useCallback, useState } from "react";
import { Project } from "types";
import { labelledSortOptions } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { LabelledSortOption } from "utils";
import { SearchProjectsPageProps } from "./types";
import { useLocation, useRoute } from "wouter";

const SORT_OPTIONS: LabelledSortOption<ProjectSortBy>[] = labelledSortOptions(ProjectSortBy);

export const SearchProjectsPage = ({
    session
}: SearchProjectsPageProps) => {
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.SearchProjects}/:id`);
    const [, setLocation] = useLocation();
    // Handles dialog when selecting a search result
    const [selected, setSelected] = useState<Project | undefined>(undefined);
    const selectedDialogOpen = Boolean(params?.id) || Boolean(selected);
    const handleSelectedDialogClose = useCallback(() => {
        console.log('handleSelectedDialogClose');
        setSelected(undefined);
        // If selected data exists, then we know we can go back to the previous page
        if (selected) {
            window.history.back();
        }
        // Otherwise the user must have entered the page directly, so we can navigate to the search page
        else {
            setLocation(APP_LINKS.SearchProjects);
        }
    }, []);

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
                title={selected?.name ?? "Project"}
                open={selectedDialogOpen}
                onClose={handleSelectedDialogClose}
            >
                <ProjectView partialData={selected} />
            </ViewDialogBase>
            {/* Search component */}
            <BaseSearchPage
                title="Search Projects"
                searchPlaceholder="Search..."
                sortOptions={SORT_OPTIONS}
                defaultSortOption={SORT_OPTIONS[1]}
                query={projectsQuery}
                listItemFactory={listItemFactory}
                getOptionLabel={(o: any) => o.name}
                onObjectSelect={(selected: Project) => setSelected(selected)}
                popupButtonText="Add"
                popupButtonTooltip="Can't find wha you're looking for? Create it!😎"
                onPopupButtonClick={handleSurpriseDialogOpen}
            />
        </>
    )
}