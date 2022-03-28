import { APP_LINKS, ROLES } from "@local/shared";
import { routineDefaultSortOption, routineOptionLabel, RoutineSortOptions, RoutineListItem, RoutineDialog, ListMenu } from "components";
import { routinesQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useState } from "react";
import { Routine } from "types";
import { Pubs } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchRoutinesPageProps } from "./types";
import { useLocation } from "wouter";
import { ListMenuItemData } from "components/dialogs/types";

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

    // Menu for picking which routine type to add
    const [addAnchor, setAddAnchor] = useState<any>(null);
    const openAdd = useCallback((ev: MouseEvent<HTMLDivElement>) => {
        const canAdd = Array.isArray(session?.roles) && session.roles.includes(ROLES.Actor);
        if (canAdd) {
            setAddAnchor(ev.currentTarget)
        }
        else {
            PubSub.publish(Pubs.Snack, { message: 'Must be logged in.', severity: 'error' });
            setLocation(`${APP_LINKS.Start}?redirect=${encodeURIComponent(APP_LINKS.SearchRoutines)}`);
        }
    }, [session?.roles, setLocation]);
    const closeAdd = useCallback(() => setAddAnchor(null), []);
    const handleAddSelect = useCallback((option: any) => {
        console.log('handleADdSelect', option);
        if (option === 'basic') setLocation(`${APP_LINKS.SearchRoutines}/add`)
        else setLocation(`${APP_LINKS.Build}/add`)
    }, []);
    const addOptions: ListMenuItemData<string>[] = [
        { label: 'Basic (Single Step)', value: 'basic' },
        { label: 'Advanced (Multi Step)', value: 'advanced' },
    ]

    const listItemFactory = (node: Routine, index: number) => (
        <RoutineListItem
            key={`routine-list-item-${index}`}
            index={index}
            session={session}
            data={node}
            onClick={(_e, selected: Routine) => setSelectedItem(selected)}
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
            {/* Select type to add */}
            <ListMenu
                id={`add-routine-select-type-menu`}
                anchorEl={addAnchor}
                title='Select Routine Type'
                data={addOptions}
                onSelect={handleAddSelect}
                onClose={closeAdd}
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
                onAddClick={openAdd}
                popupButtonText="Add"
                popupButtonTooltip="Can't find what you're looking for? Create it!😎"
                onPopupButtonClick={openAdd}
            />
        </>
    )
}