import { APP_LINKS, ROLES } from "@local/shared";
import { projectDefaultSortOption, ProjectSortOptions, ProjectDialog } from "components";
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
    }, []);
    useEffect(() => {
        if (selectedItem) {
            setLocation(`${APP_LINKS.SearchProjects}/view/${selectedItem.id}`);
        }
    }, [selectedItem, setLocation]);
    useEffect(() => {
        if (location === APP_LINKS.SearchProjects) {
            setSelectedItem(undefined);
        }
    }, [location])

    // Handles dialog when adding a new organization
    const handleAddDialogOpen = useCallback(() => {
        const canAdd = Array.isArray(session?.roles) && session.roles.includes(ROLES.Actor);
        if (canAdd) {
            setLocation(`${APP_LINKS.SearchProjects}/add`)
        }
        else {
            PubSub.publish(Pubs.Snack, { message: 'Must be logged in.', severity: 'error' });
            setLocation(`${APP_LINKS.Start}?redirect=${encodeURIComponent(APP_LINKS.SearchProjects)}`);
        }
    }, [session?.roles, setLocation]);

    return (
        <>
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
                itemKeyPrefix="project-list-item"
                title="Projects"
                searchPlaceholder="Search..."
                sortOptions={ProjectSortOptions}
                defaultSortOption={projectDefaultSortOption}
                query={projectsQuery}
                onObjectSelect={handleSelected}
                onAddClick={handleAddDialogOpen}
                popupButtonText="Add"
                popupButtonTooltip="Can't find wha you're looking for? Create it!😎"
                onPopupButtonClick={handleAddDialogOpen}
                session={session}
            />
        </>
    )
}