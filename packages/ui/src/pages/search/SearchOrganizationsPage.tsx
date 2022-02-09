import { APP_LINKS, ROLES } from "@local/shared";
import { organizationDefaultSortOption, OrganizationListItem, organizationOptionLabel, OrganizationSortOptions, ShareDialog } from "components";
import { OrganizationDialog } from "components/dialogs/OrganizationDialog/OrganizationDialog";
import { organizationsQuery } from "graphql/query";
import { useCallback, useEffect, useState } from "react";
import { Organization } from "types";
import { Pubs } from "utils";
import { useLocation } from "wouter";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchOrganizationsPageProps } from "./types";

export const SearchOrganizationsPage = ({
    session
}: SearchOrganizationsPageProps) => {
    const [location, setLocation] = useLocation();

    // Handles item add/select/edit
    const [selectedItem, setSelectedItem] = useState<Organization | undefined>(undefined);
    const handleSelected = useCallback((selected: Organization) => {
        setSelectedItem(selected);
    }, [setLocation]);
    useEffect(() => {
        if (selectedItem) {
            setLocation(`${APP_LINKS.SearchOrganizations}/view/${selectedItem.id}`);
        }
    }, [selectedItem, setLocation]);
    useEffect(() => {
        if (location === APP_LINKS.SearchOrganizations) {
            setSelectedItem(undefined);
        }
    }, [location])

    // Handles dialog when adding a new organization
    const handleAddDialogOpen = useCallback(() => {
        const canAdd = Array.isArray(session?.roles) && session.roles.includes(ROLES.Actor);
        console.log('handleAddDialogOpen', canAdd, session);
        if (canAdd) {
            setLocation(`${APP_LINKS.SearchOrganizations}/add`)
        }
        else {
            PubSub.publish(Pubs.Snack, { message: 'Must be logged in.', severity: 'error' });
            setLocation(APP_LINKS.Start)
        }
    }, [session, setLocation]);

    // Handles dialog for the button that appears after scrolling a certain distance
    const [surpriseDialogOpen, setSurpriseDialogOpen] = useState(false);
    const handleSurpriseDialogOpen = useCallback(() => setSurpriseDialogOpen(true), []);
    const handleSurpriseDialogClose = useCallback(() => setSurpriseDialogOpen(false), []);

    const listItemFactory = (node: Organization, index: number) => (
        <OrganizationListItem
            key={`organization-list-item-${index}`}
            index={index}
            session={session}
            data={node}
            isOwn={false}
            onClick={(selected: Organization) => setSelectedItem(selected)}
        />)

    return (
        <>
            {/* Invite link dialog */}
            <ShareDialog onClose={handleSurpriseDialogClose} open={surpriseDialogOpen} />
            {/* View/Add/Update dialog */}
            <OrganizationDialog
                hasPrevious={false}
                hasNext={false}
                canEdit={false}
                partialData={selectedItem}
                session={session}
            />
            {/* Search component */}
            <BaseSearchPage
                title="Organizations"
                searchPlaceholder="Search..."
                sortOptions={OrganizationSortOptions}
                defaultSortOption={organizationDefaultSortOption}
                query={organizationsQuery}
                listItemFactory={listItemFactory}
                getOptionLabel={organizationOptionLabel}
                onObjectSelect={handleSelected}
                onAddClick={handleAddDialogOpen}
                popupButtonText="Invite"
                popupButtonTooltip="Can't find who you're looking for? Invite them😊"
                onPopupButtonClick={handleSurpriseDialogOpen}
            />
        </>
    )
}