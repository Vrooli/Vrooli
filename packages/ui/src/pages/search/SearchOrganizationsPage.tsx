import { APP_LINKS } from "@local/shared";
import { AddDialogBase, organizationDefaultSortOption, OrganizationListItem, organizationOptionLabel, OrganizationSortOptions, OrganizationView, ShareDialog, ViewDialogBase } from "components";
import { organizationsQuery } from "graphql/query";
import { useCallback, useMemo, useState } from "react";
import { Organization } from "types";
import { useLocation, useRoute } from "wouter";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchOrganizationsPageProps } from "./types";

export const SearchOrganizationsPage = ({
    session
}: SearchOrganizationsPageProps) => {
    const [, setLocation] = useLocation();
    const [match, params] = useRoute(`${APP_LINKS.SearchOrganizations}/:id`);

    // Handles dialog when selecting a search result
    const [selected, setSelected] = useState<Organization | undefined>(undefined);
    const selectedDialogOpen = Boolean(match || selected);
    const handleSelected = useCallback((selected: Organization) => {
        setSelected(selected);
        setLocation(`${APP_LINKS.SearchOrganizations}/${selected.id}`);
    }, [setLocation]);
    const handleSelectedDialogClose = useCallback(() => {
        setSelected(undefined);
        // If selected data exists, then we know we can go back to the previous page
        if (selected) window.history.back();
        // Otherwise the user must have entered the page directly, so we can navigate to the search page
        else setLocation(APP_LINKS.SearchOrganizations);
    }, [setLocation, selected]);

    // Handles dialog when adding a new organization
    const [addDialogOpen, setAddDialogOpen] = useState(false);
    const handleAddDialogOpen = useCallback(() => setAddDialogOpen(true), [setAddDialogOpen]);
    const handleAddDialogClose = useCallback(() => setAddDialogOpen(false), [setAddDialogOpen]);
    const handleAddDialogSubmit = useCallback((organization: Organization) => {
        // TODO
    }, [setAddDialogOpen]);

    const partialData = useMemo(() => {
        if (selected) return selected;
        if (params?.id) return { id: params.id };
        return undefined;
    }, [params, selected]);

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
            onClick={(selected: Organization) => setSelected(selected)}
        />)

    return (
        <>
            {/* Invite link dialog */}
            <ShareDialog onClose={handleSurpriseDialogClose} open={surpriseDialogOpen} />
            {/* Selected dialog */}
            <ViewDialogBase
                title='View Organization'
                open={selectedDialogOpen}
                onClose={handleSelectedDialogClose}
            >
                <OrganizationView session={session} partialData={partialData} />
            </ViewDialogBase>
            {/* Add dialog */}
            <AddDialogBase
                title='Add Organization'
                open={addDialogOpen}
                onClose={handleAddDialogClose}
                onSubmit={handleAddDialogSubmit}
            >
                <OrganizationModify />
            </AddDialogBase>
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