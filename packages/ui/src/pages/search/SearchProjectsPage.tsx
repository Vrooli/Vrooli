import { APP_LINKS, ROLES } from "@local/shared";
import { projectDefaultSortOption, ProjectListItem, projectOptionLabel, ProjectSortOptions, ShareDialog, ProjectDialog } from "components";
import { projectsQuery } from "graphql/query";
import { useCallback, useEffect, useState } from "react";
import { Project } from "types";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchProjectsPageProps } from "./types";
import { useLocation } from "wouter";
import { Pubs } from "utils";

export const SearchProjectsPage = ({
    session
}: SearchProjectsPageProps) => {
    const [location, setLocation] = useLocation();

    // Handles item add/select/edit
    const [selectedItem, setSelectedItem] = useState<Project | undefined>(undefined);
    const handleSelected = useCallback((selected: Project) => {
        setSelectedItem(selected);
    }, [setLocation]);
    useEffect(() => {
        if (selectedItem) {
            setLocation(`${APP_LINKS.SearchProjects}/view/${selectedItem.id}`, { replace: true });
        }
    }, [selectedItem, setLocation]);
    useEffect(() => {
        if (location === APP_LINKS.SearchProjects) {
            setSelectedItem(undefined);
        }
    }, [location])

    // Handles dialog when adding a new organization
    const handleAddDialogOpen = useCallback(() => {
        const canAdd = Array.isArray(session?.roles) && !session.roles.includes(ROLES.Actor);
        if (canAdd) {
            setLocation(`${APP_LINKS.SearchProjects}/add`)
        }
        else {
            PubSub.publish(Pubs.Snack, { message: 'Must be logged in.', severity: 'error' });
            setLocation(APP_LINKS.Start)
        }
    }, [setLocation]);

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
            onClick={(selected: Project) => setSelectedItem(selected)}
        />)

    return (
        <>
            {/* Invite link dialog */}
            <ShareDialog onClose={handleSurpriseDialogClose} open={surpriseDialogOpen} />
            {/* Selected dialog */}
            <ProjectDialog
                hasPrevious={false}
                hasNext={false}
                canEdit={false}
                partialData={selectedItem}
                session={session}
            />
            {/* Search component */}
            <BaseSearchPage
                title="Projects"
                searchPlaceholder="Search..."
                sortOptions={ProjectSortOptions}
                defaultSortOption={projectDefaultSortOption}
                query={projectsQuery}
                listItemFactory={listItemFactory}
                getOptionLabel={projectOptionLabel}
                onObjectSelect={handleSelected}
                onAddClick={handleAddDialogOpen}
                popupButtonText="Add"
                popupButtonTooltip="Can't find wha you're looking for? Create it!😎"
                onPopupButtonClick={handleSurpriseDialogOpen}
            />
        </>
    )
}