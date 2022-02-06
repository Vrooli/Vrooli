import { StandardListItem, StandardView, ShareDialog, BaseObjectDialog, StandardSortOptions, standardDefaultSortOption, standardOptionLabel, StandardDialog } from "components";
import { standardsQuery } from "graphql/query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Standard } from "types";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchStandardsPageProps } from "./types";
import { useLocation, useRoute } from "wouter";
import { APP_LINKS, ROLES } from "@local/shared";
import { Pubs } from "utils";

export const SearchStandardsPage = ({
    session
}: SearchStandardsPageProps) => {
    const [location, setLocation] = useLocation();

    // Handles item add/select/edit
    const [selectedItem, setSelectedItem] = useState<Standard | undefined>(undefined);
    const handleSelected = useCallback((selected: Standard) => {
        setSelectedItem(selected);
    }, [setLocation]);
    useEffect(() => {
        if (selectedItem) {
            setLocation(`${APP_LINKS.SearchStandards}/view/${selectedItem.id}`, { replace: true });
        }
    }, [selectedItem, setLocation]);
    useEffect(() => {
        if (location === APP_LINKS.SearchStandards) {
            setSelectedItem(undefined);
        }
    }, [location])

    // Handles dialog when adding a new organization
    const handleAddDialogOpen = useCallback(() => {
        const canAdd = Array.isArray(session?.roles) && session.roles.includes(ROLES.Actor);
        if (canAdd) {
            setLocation(`${APP_LINKS.SearchStandards}/add`)
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

    const listItemFactory = (node: Standard, index: number) => (
        <StandardListItem
            key={`standard-list-item-${index}`}
            session={session}
            data={node}
            isOwn={false}
            onClick={(selected: Standard) => setSelectedItem(selected)}
        />)

    return (
        <>
            {/* Invite link dialog */}
            <ShareDialog onClose={handleSurpriseDialogClose} open={surpriseDialogOpen} />
            {/* Selected dialog */}
            <StandardDialog
                hasPrevious={false}
                hasNext={false}
                canEdit={false}
                partialData={selectedItem}
                session={session}
            />
            {/* Search component */}
            <BaseSearchPage
                title="Standards"
                searchPlaceholder="Search..."
                sortOptions={StandardSortOptions}
                defaultSortOption={standardDefaultSortOption}
                query={standardsQuery}
                listItemFactory={listItemFactory}
                getOptionLabel={standardOptionLabel}
                onObjectSelect={handleSelected}
                onAddClick={handleAddDialogOpen}
                popupButtonText="Add"
                popupButtonTooltip="Can't find what you're looking for? Create it!😎"
                onPopupButtonClick={handleSurpriseDialogOpen}
            />
        </>
    )
}