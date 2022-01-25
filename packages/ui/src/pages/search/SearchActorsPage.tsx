import { APP_LINKS } from "@local/shared";
import { actorDefaultSortOption, ActorListItem, actorOptionLabel, ActorSortOptions, ActorView, ShareDialog, ViewDialogBase } from "components";
import { usersQuery } from "graphql/query";
import { useCallback, useMemo, useState } from "react";
import { User } from "types";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchActorsPageProps } from "./types";
import { useLocation, useRoute } from "wouter";

export const SearchActorsPage = ({
    session
}: SearchActorsPageProps) => {
    const [, setLocation] = useLocation();
    const [match, params] = useRoute(`${APP_LINKS.SearchUsers}/:id`);
    // Handles dialog when selecting a search result
    const [selected, setSelected] = useState<User | undefined>(undefined);
    const selectedDialogOpen = Boolean(match || selected);
    const handleSelected = useCallback((selected: User) => {
        setSelected(selected);
        setLocation(`${APP_LINKS.SearchUsers}/${selected.id}`);
    }, [setLocation]);
    const handleSelectedDialogClose = useCallback(() => {
        console.log("handleSelectedDialogClose");
        setSelected(undefined);
        // If selected data exists, then we know we can go back to the previous page
        if (selected) window.history.back();
        // Otherwise the user must have entered the page directly, so we can navigate to the search page
        else setLocation(APP_LINKS.SearchUsers);
    }, [setLocation, selected]);

    const partialData = useMemo(() => {
        if (selected) return selected;
        if (params?.id) return { id: params.id };
        return undefined;
    }, [params, selected]);

    // Handles dialog for the button that appears after scrolling a certain distance
    const [surpriseDialogOpen, setSurpriseDialogOpen] = useState(false);
    const handleSurpriseDialogOpen = useCallback(() => setSurpriseDialogOpen(true), []);
    const handleSurpriseDialogClose = useCallback(() => setSurpriseDialogOpen(false), []);

    const listItemFactory = (node: User, index: number) => (
        <ActorListItem
            key={`actor-list-item-${index}`}
            session={session}
            data={node}
            isOwn={session?.id === node.id}
            onClick={handleSelected}
        />)

    return (
        <>
            {/* Invite link dialog */}
            <ShareDialog onClose={handleSurpriseDialogClose} open={surpriseDialogOpen} />
            {/* Selected dialog */}
            <ViewDialogBase
                title='View User'
                open={selectedDialogOpen}
                onClose={handleSelectedDialogClose}
            >
                <ActorView session={session} partialData={partialData} />
            </ViewDialogBase>
            {/* Search component */}
            <BaseSearchPage
                title="Search Actors"
                searchPlaceholder="Search by username..."
                sortOptions={ActorSortOptions}
                defaultSortOption={actorDefaultSortOption}
                query={usersQuery}
                listItemFactory={listItemFactory}
                getOptionLabel={actorOptionLabel}
                onObjectSelect={handleSelected}
                popupButtonText="Invite"
                popupButtonTooltip="Can't find who you're looking for? Invite them😊"
                onPopupButtonClick={handleSurpriseDialogOpen}
            />
        </>
    )
}