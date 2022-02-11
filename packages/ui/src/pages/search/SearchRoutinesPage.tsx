import { APP_LINKS, ROLES } from "@local/shared";
import { routineDefaultSortOption, routineOptionLabel, RoutineSortOptions, RoutineListItem, RoutineDialog } from "components";
import { routinesQuery } from "graphql/query";
import { useCallback, useEffect, useState } from "react";
import { Routine } from "types";
import { Pubs } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchRoutinesPageProps } from "./types";
import { useLocation } from "wouter";

export const SearchRoutinesPage = ({
    session
}: SearchRoutinesPageProps) => {
    const [location, setLocation] = useLocation();

    // Handles item add/select/edit
    const [selectedItem, setSelectedItem] = useState<Routine | undefined>(undefined);
    const handleSelected = useCallback((selected: Routine) => {
        setSelectedItem(selected);
    }, []);
    useEffect(() => {
        if (selectedItem) {
            setLocation(`${APP_LINKS.SearchRoutines}/view/${selectedItem.id}`);
        }
    }, [selectedItem, setLocation]);
    useEffect(() => {
        if (location === APP_LINKS.SearchRoutines) {
            setSelectedItem(undefined);
        }
    }, [location])

    // Handles dialog when adding a new organization
    const handleAddDialogOpen = useCallback(() => {
        const canAdd = Array.isArray(session?.roles) && session.roles.includes(ROLES.Actor);
        if (canAdd) {
            setLocation(`${APP_LINKS.SearchRoutines}/add`)
        }
        else {
            PubSub.publish(Pubs.Snack, { message: 'Must be logged in.', severity: 'error' });
            setLocation(APP_LINKS.Start)
        }
    }, [session?.roles, setLocation]);

    const listItemFactory = (node: Routine, index: number) => (
        <RoutineListItem
            key={`routine-list-item-${index}`}
            index={index}
            session={session}
            data={node}
            onClick={(selected: Routine) => setSelectedItem(selected)}
        />)

    return (
        <>
            {/* Selected dialog */}
            <RoutineDialog
                hasPrevious={false}
                hasNext={false}
                canEdit={false}
                partialData={selectedItem}
                session={session}
            />
            {/* Search component */}
            <BaseSearchPage
                title="Routines"
                searchPlaceholder="Search..."
                sortOptions={RoutineSortOptions}
                defaultSortOption={routineDefaultSortOption}
                query={routinesQuery}
                listItemFactory={listItemFactory}
                getOptionLabel={routineOptionLabel}
                onObjectSelect={handleSelected}
                onAddClick={handleAddDialogOpen}
                popupButtonText="Add"
                popupButtonTooltip="Can't find what you're looking for? Create it!😎"
                onPopupButtonClick={handleAddDialogOpen}
            />
        </>
    )
}