import { UserSortBy } from "@local/shared";
import { ActorListItem, ActorView, ShareDialog, ViewDialogBase } from "components";
import { usersQuery } from "graphql/query";
import { useCallback, useState } from "react";
import { User } from "types";
import { labelledSortOptions } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { LabelledSortOption } from "utils";
import { SearchActorsPageProps } from "./types";

const SORT_OPTIONS: LabelledSortOption<UserSortBy>[] = labelledSortOptions(UserSortBy);

export const SearchActorsPage = ({
    session
}: SearchActorsPageProps) => {
    // Handles dialog when selecting a search result
    const [selected, setSelected] = useState<User | undefined>(undefined);
    const selectedDialogOpen = Boolean(selected);
    const handleSelectedDialogClose = useCallback(() => setSelected(undefined), []);

    // Handles dialog for the button that appears after scrolling a certain distance
    const [surpriseDialogOpen, setSurpriseDialogOpen] = useState(false);
    const handleSurpriseDialogOpen = useCallback(() => setSurpriseDialogOpen(true), []);
    const handleSurpriseDialogClose = useCallback(() => setSurpriseDialogOpen(false), []);

    const listItemFactory = (node: User, index: number) => (
        <ActorListItem
            key={`actor-list-item-${index}`}
            session={session}
            data={node}
            isOwn={false}
            onClick={(selected: User) => setSelected(selected)}
        />)

    return (
        <>
            {/* Invite link dialog */}
            <ShareDialog onClose={handleSurpriseDialogClose} open={surpriseDialogOpen} />
            {/* Selected dialog */}
            <ViewDialogBase
                title={selected?.username ?? "User"}
                open={selectedDialogOpen}
                onClose={handleSelectedDialogClose}
            >
                <ActorView partialData={selected} />
            </ViewDialogBase>
            {/* Search component */}
            <BaseSearchPage
                title="Search Actors"
                searchPlaceholder="Search by username..."
                sortOptions={SORT_OPTIONS}
                defaultSortOption={SORT_OPTIONS[1]}
                query={usersQuery}
                listItemFactory={listItemFactory}
                getOptionLabel={(o: any) => o.username}
                onObjectSelect={(selected: User) => setSelected(selected)}
                popupButtonText="Invite"
                popupButtonTooltip="Can't find who you're looking for? Invite them😊"
                onPopupButtonClick={handleSurpriseDialogOpen}
            />
        </>
    )
}