import { APP_LINKS } from "@local/shared";
import { organizationDefaultSortOption, OrganizationListItem, organizationOptionLabel, OrganizationSortOptions, ShareDialog } from "components";
import { OrganizationDialog } from "components/dialogs/OrganizationDialog/OrganizationDialog";
import { organizationsQuery } from "graphql/query";
import { useCallback, useEffect, useState } from "react";
import { Organization } from "types";
import { useLocation } from "wouter";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchOrganizationsPageProps } from "./types";

export const SearchOrganizationsPage = ({
    session
}: SearchOrganizationsPageProps) => {
    const [location, setLocation] = useLocation();

    console.log("SearchOrganizationsPage", window);

    // Handles item add/select/edit
    const [selectedItem, setSelectedItem] = useState<Organization | undefined>(undefined);
    useEffect(() => {
        if (selectedItem) {
            setLocation(`${APP_LINKS.SearchOrganizations}/view/${selectedItem.id}`, { replace: true });
        }
    }, [selectedItem, setLocation]);
    useEffect(() => {
        if (location === APP_LINKS.SearchOrganizations) {
            setSelectedItem(undefined);
        }
    }, [location])

    // Handles dialog when adding a new organization
    const handleAddDialogOpen = useCallback(() => setLocation(`${APP_LINKS.SearchOrganizations}/add`), [setLocation]);

    // Handles dialog for the button that appears after scrolling a certain distance
    const [surpriseDialogOpen, setSurpriseDialogOpen] = useState(false);
    const handleSurpriseDialogOpen = useCallback(() => setSurpriseDialogOpen(true), []);
    const handleSurpriseDialogClose = useCallback(() => setSurpriseDialogOpen(false), []);

    const listItemFactory = (node: Organization, index: number) => (
        <OrganizationListItem
            key={`organization-list-item-${index}`}
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
                getOptionLabel={(o: any) => o.name}
                onObjectSelect={organizationOptionLabel}
                showAddButton={true}
                onAddClick={handleAddDialogOpen}
                popupButtonText="Invite"
                popupButtonTooltip="Can't find who you're looking for? Invite them😊"
                onPopupButtonClick={handleSurpriseDialogOpen}
            />
        </>
    )
}