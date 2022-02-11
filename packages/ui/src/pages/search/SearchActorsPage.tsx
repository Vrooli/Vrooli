import { APP_LINKS } from "@local/shared";
import { actorDefaultSortOption, ActorListItem, actorOptionLabel, ActorSortOptions, ShareDialog, UserDialog } from "components";
import { usersQuery } from "graphql/query";
import { useCallback, useEffect, useState } from "react";
import { User } from "types";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchActorsPageProps } from "./types";
import { useLocation } from "wouter";

export const SearchActorsPage = ({
    session
}: SearchActorsPageProps) => {
    const [location, setLocation] = useLocation();

    // Handles item add/select/edit
    const [selectedItem, setSelectedItem] = useState<User | undefined>(undefined);
    const handleSelected = useCallback((selected: User) => {
        setSelectedItem(selected);
    }, [setLocation]);
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

    const listItemFactory = (node: User, index: number) => (
        <ActorListItem
            key={`actor-list-item-${index}`}
            index={index}
            session={session}
            data={node}
            onClick={(selected: User) => setSelectedItem(selected)}
        />)

    return (
        <>
            {/* Invite link dialog */}
            <ShareDialog onClose={handleSurpriseDialogClose} open={surpriseDialogOpen} />
            {/* Selected dialog */}
            <UserDialog
                hasPrevious={false}
                hasNext={false}
                canEdit={false}
                partialData={selectedItem}
                session={session}
            />
            {/* Search component */}
            <BaseSearchPage
                title="Users"
                searchPlaceholder="Search by username..."
                sortOptions={ActorSortOptions}
                defaultSortOption={actorDefaultSortOption}
                query={usersQuery}
                listItemFactory={listItemFactory}
                getOptionLabel={actorOptionLabel}
                onObjectSelect={handleSelected}
                showAddButton={false}
                popupButtonText="Invite"
                popupButtonTooltip="Can't find who you're looking for? Invite them😊"
                onPopupButtonClick={handleSurpriseDialogOpen}
            />
        </>
    )
}