import { APP_LINKS } from "@local/shared";
import { RoutineDialog, ListMenu } from "components";
import { routinesQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useState } from "react";
import { Routine } from "types";
import { ObjectType, PubSub, stringifySearchParams } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchRoutinesPageProps } from "./types";
import { useLocation } from "wouter";
import { ListMenuItemData } from "components/dialogs/types";
import { validate as uuidValidate } from "uuid";

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
        const loggedIn = session?.isLoggedIn === true && uuidValidate(session?.id ?? '');
        if (loggedIn) {
            setAddAnchor(ev.currentTarget)
        }
        else {
            PubSub.get().publishSnack({ message: 'Must be logged in.', severity: 'error' });
            setLocation(`${APP_LINKS.Start}${stringifySearchParams({
                redirect: APP_LINKS.SearchRoutines
            })}`);
        }
    }, [session?.id, session?.isLoggedIn, setLocation]);
    const closeAdd = useCallback(() => setAddAnchor(null), []);
    const handleAddSelect = useCallback((option: any) => {
        if (option === 'basic') setLocation(`${APP_LINKS.SearchRoutines}/add`)
        else setLocation(`${APP_LINKS.Routine}/add?build=true`)
    }, [setLocation]);
    const addOptions: ListMenuItemData<string>[] = [
        { label: 'Basic (Single Step)', value: 'basic' },
        { label: 'Advanced (Multi Step)', value: 'advanced' },
    ]

    return (
        <>
            {/* Selected dialog */}
            <RoutineDialog
                partialData={selectedItem}
                session={session}
                zIndex={200}
            />
            {/* Select type to add */}
            <ListMenu
                id={`add-routine-select-type-menu`}
                anchorEl={addAnchor}
                title='Select Routine Type'
                data={addOptions}
                onSelect={handleAddSelect}
                onClose={closeAdd}
                zIndex={200}
            />
            {/* Search component */}
            <BaseSearchPage
                itemKeyPrefix="routine-list-item"
                title="Routines"
                searchPlaceholder="Search..."
                query={routinesQuery}
                objectType={ObjectType.Routine}
                onObjectSelect={handleSelected}
                onAddClick={openAdd}
                popupButtonText="Add"
                popupButtonTooltip="Can't find what you're looking for? Create it!😎"
                onPopupButtonClick={openAdd}
                session={session}
                where={{ isComplete: true, isInternal: false }}
            />
        </>
    )
}