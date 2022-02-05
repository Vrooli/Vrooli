import { APP_LINKS, ROLES } from "@local/shared";
import { routineDefaultSortOption, routineOptionLabel, RoutineSortOptions, RoutineListItem, ShareDialog, RoutineDialog } from "components";
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
    }, [setLocation]);
    useEffect(() => {
        if (selectedItem) {
            setLocation(`${APP_LINKS.SearchRoutines}/view/${selectedItem.id}`, { replace: true });
        }
    }, [selectedItem, setLocation]);
    useEffect(() => {
        if (location === APP_LINKS.SearchRoutines) {
            setSelectedItem(undefined);
        }
    }, [location])

    // Handles dialog when adding a new organization
    const handleAddDialogOpen = useCallback(() => {
        const canAdd = Array.isArray(session?.roles) && !session.roles.includes(ROLES.Actor);
        if (canAdd) {
            setLocation(`${APP_LINKS.SearchRoutines}/add`)
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

    const listItemFactory = (node: Routine, index: number) => (
        <RoutineListItem
            key={`routine-list-item-${index}`}
            session={session}
            data={node}
            isOwn={false}
            onClick={(selected: Routine) => setSelectedItem(selected)}
        />)

    return (
        <>
            {/* Invite link dialog */}
            <ShareDialog onClose={handleSurpriseDialogClose} open={surpriseDialogOpen} />
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
                onPopupButtonClick={handleSurpriseDialogOpen}
            />
        </>
    )
}