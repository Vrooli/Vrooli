import { UserSortBy } from "@local/shared";
import { Dialog, DialogTitle } from "@mui/material";
import { ActorListItem, AddDialogBase, ShareDialog } from "components";
import { usersQuery } from "graphql/query";
import { ActorViewPage } from "pages";
import { useCallback, useState } from "react";
import { User } from "types";
import { SortValueToLabelMap } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchSortBy } from "./types";

const SORT_OPTIONS: SearchSortBy<UserSortBy>[] = Object.values(UserSortBy).map((sortOption) => ({
    label: SortValueToLabelMap[sortOption],
    value: sortOption as UserSortBy
}));

export const SearchActorsPage = () => {
    const [selected, setSelected] = useState<User | undefined>(undefined);
    const selectedDialogOpen = Boolean(selected);
    const [inviteDialogOpen, setInviteDialogOpen] = useState(false);

    const handleSelectedDialogClose = useCallback(() => setSelected(undefined), []);
    const handleInviteDialogOpen = useCallback(() => setInviteDialogOpen(true), []);
    const handleInviteDialogClose = useCallback(() => setInviteDialogOpen(false), []);

    const listItemFactory = (node: User, index: number) => (
        <ActorListItem
            key={`actor-list-item-${index}`}
            data={node}
            isStarred={false}
            isOwn={false}
            onClick={(selected: User) => setSelected(selected)}
            onStarClick={() => { }}
        />)

    return (
        <>
            {/* Invite link dialog */}
            <ShareDialog onClose={handleInviteDialogClose} open={inviteDialogOpen} />
            {/* Selected dialog */}
            <AddDialogBase
                title={selected?.username ?? "User"}
                open={selectedDialogOpen}
                onClose={handleSelectedDialogClose}
                onSubmit={() => { }}
            >
                {/* <ActorViewPage data={selected} /> */}
            </AddDialogBase>
            {/* Search component */}
            <BaseSearchPage
                title={'Search Actors'}
                searchPlaceholder="Search by username..."
                sortOptions={SORT_OPTIONS}
                defaultSortOption={SORT_OPTIONS[1]}
                query={usersQuery}
                listItemFactory={listItemFactory}
                getOptionLabel={(o: any) => o.username}
                onObjectSelect={(selected: User) => setSelected(selected)}
                popupButtonText={'Invite'}
                popupButtonTooltip={"Can't find who you're looking for? Invite them😊"}
                onPopupButtonClick={handleInviteDialogOpen}
            />
        </>
    )
}