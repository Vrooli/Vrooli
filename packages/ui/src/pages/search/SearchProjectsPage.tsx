import { APP_LINKS, ROLES } from "@local/shared";
import { ProjectDialog } from "components";
import { projectsQuery } from "graphql/query";
import { useCallback, useEffect, useState } from "react";
import { Project } from "types";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchProjectsPageProps } from "./types";
import { useLocation } from "wouter";
import { ObjectType, Pubs, stringifySearchParams } from "utils";

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
            setLocation(`${APP_LINKS.Start}${stringifySearchParams({
                redirect: APP_LINKS.SearchProjects
            })}`);
        }
    }, [session?.roles, setLocation]);

    return (
        <>
            {/* Selected dialog */}
            <ProjectDialog
                partialData={selectedItem}
                session={session}
                zIndex={200}
            />
            {/* Search component */}
            <BaseSearchPage
                itemKeyPrefix="project-list-item"
                title="Projects"
                searchPlaceholder="Search..."
                query={projectsQuery}
                objectType={ObjectType.Project}
                onObjectSelect={handleSelected}
                onAddClick={handleAddDialogOpen}
                popupButtonText="Add"
                popupButtonTooltip="Can't find wha you're looking for? Create it!😎"
                onPopupButtonClick={handleAddDialogOpen}
                session={session}
                where={{ isComplete: true }}
            />
        </>
    )
}