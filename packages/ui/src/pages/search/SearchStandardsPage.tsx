import { StandardDialog } from "components";
import { standardsQuery } from "graphql/query";
import { useCallback, useEffect, useState } from "react";
import { Standard } from "types";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchStandardsPageProps } from "./types";
import { useLocation } from "wouter";
import { APP_LINKS } from "@local/shared";
import { ObjectType, PubSub, stringifySearchParams } from "utils";
import { validate as uuidValidate } from 'uuid';

export const SearchStandardsPage = ({
    session
}: SearchStandardsPageProps) => {
    const [location, setLocation] = useLocation();

    // Handles item add/select/edit
    const [selectedItem, setSelectedItem] = useState<Standard | undefined>(undefined);
    const handleSelected = useCallback((selected: Standard) => {
        setSelectedItem(selected);
    }, []);
    useEffect(() => {
        if (selectedItem) {
            setLocation(`${APP_LINKS.SearchStandards}/view/${selectedItem.id}`);
        }
    }, [selectedItem, setLocation]);
    useEffect(() => {
        if (location === APP_LINKS.SearchStandards) {
            setSelectedItem(undefined);
        }
    }, [location])

    // Handles dialog when adding a new organization
    const handleAddDialogOpen = useCallback(() => {
        const loggedIn = session?.isLoggedIn === true && uuidValidate(session?.id ?? '');
        if (loggedIn) {
            setLocation(`${APP_LINKS.SearchStandards}/add`)
        }
        else {
            PubSub.get().publishSnack({ message: 'Must be logged in.', severity: 'error' });
            setLocation(`${APP_LINKS.Start}${stringifySearchParams({
                redirect: APP_LINKS.SearchStandards
            })}`);
        }
    }, [session?.id, session?.isLoggedIn, setLocation]);

    return (
        <>
            {/* Selected dialog */}
            <StandardDialog
                partialData={selectedItem}
                session={session}
                zIndex={200}
            />
            {/* Search component */}
            <BaseSearchPage
                itemKeyPrefix="standard-list-item"
                title="Standards"
                searchPlaceholder="Search..."
                query={standardsQuery}
                objectType={ObjectType.Standard}
                onObjectSelect={handleSelected}
                onAddClick={handleAddDialogOpen}
                popupButtonText="Add"
                popupButtonTooltip="Can't find what you're looking for? Create it!😎"
                onPopupButtonClick={handleAddDialogOpen}
                session={session}
                where={{ type: 'JSON' }}
            />
        </>
    )
}