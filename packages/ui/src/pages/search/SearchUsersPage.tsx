import { APP_LINKS } from "@local/shared";
import { ShareDialog, UserDialog } from "components";
import { usersQuery } from "graphql/query";
import { useCallback, useEffect, useState } from "react";
import { User } from "types";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchUsersPageProps } from "./types";
import { useLocation } from "wouter";
import { ObjectType } from "utils";

export const SearchUsersPage = ({
    session
}: SearchUsersPageProps) => {
    const [location, setLocation] = useLocation();

    // Handles item add/select/edit
    const [selectedItem, setSelectedItem] = useState<User | undefined>(undefined);
    const handleSelected = useCallback((selected: User) => {
        setSelectedItem(selected);
    }, []);
    useEffect(() => {
        if (selectedItem) {
            setLocation(`${APP_LINKS.SearchUsers}/view/${selectedItem.id}`);
        }
    }, [selectedItem, setLocation]);
    useEffect(() => {
        if (location === APP_LINKS.SearchUsers) {
            setSelectedItem(undefined);
        }
    }, [location])

    // Handles dialog for the button that appears after scrolling a certain distance
    const [surpriseDialogOpen, setSurpriseDialogOpen] = useState(false);
    const handleSurpriseDialogOpen = useCallback(() => setSurpriseDialogOpen(true), []);
    const handleSurpriseDialogClose = useCallback(() => setSurpriseDialogOpen(false), []);

    return (
        <>
            {/* Invite link dialog */}
            <ShareDialog 
                onClose={handleSurpriseDialogClose} 
                open={surpriseDialogOpen} 
                zIndex={200}
            />
            {/* Selected dialog */}
            <UserDialog
                partialData={selectedItem}
                session={session}
                zIndex={200}
            />
            {/* Search component */}
            <BaseSearchPage
                itemKeyPrefix="user-list-item"
                title="Users"
                searchPlaceholder="Search by name/handle..."
                query={usersQuery}
                objectType={ObjectType.User}
                onObjectSelect={handleSelected}
                showAddButton={false}
                popupButtonText="Invite"
                popupButtonTooltip="Can't find who you're looking for? Invite them😊"
                onPopupButtonClick={handleSurpriseDialogOpen}
                session={session}
            />
        </>
    )
}