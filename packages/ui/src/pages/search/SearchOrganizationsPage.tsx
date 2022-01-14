import { OrganizationSortBy } from "@local/shared";
import { OrganizationListItem, OrganizationView, ShareDialog, ViewDialogBase } from "components";
import { organizationsQuery } from "graphql/query";
import { useCallback, useState } from "react";
import { Organization } from "types";
import { labelledSortOptions } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { LabelledSortOption } from "utils";

const SORT_OPTIONS: LabelledSortOption<OrganizationSortBy>[] = labelledSortOptions(OrganizationSortBy);

export const SearchOrganizationsPage = () => {
    // Handles dialog when selecting a search result
    const [selected, setSelected] = useState<Organization | undefined>(undefined);
    const selectedDialogOpen = Boolean(selected);
    const handleSelectedDialogClose = useCallback(() => setSelected(undefined), []);

    // Handles dialog for the button that appears after scrolling a certain distance
    const [surpriseDialogOpen, setSurpriseDialogOpen] = useState(false);
    const handleSurpriseDialogOpen = useCallback(() => setSurpriseDialogOpen(true), []);
    const handleSurpriseDialogClose = useCallback(() => setSurpriseDialogOpen(false), []);

    const listItemFactory = (node: Organization, index: number) => (
        <OrganizationListItem
            key={`organization-list-item-${index}`}
            data={node}
            isStarred={false}
            isOwn={false}
            onClick={(selected: Organization) => setSelected(selected)}
            onStarClick={() => { }}
        />)

    return (
        <>
            {/* Invite link dialog */}
            <ShareDialog onClose={handleSurpriseDialogClose} open={surpriseDialogOpen} />
            {/* Selected dialog */}
            <ViewDialogBase
                title={selected?.name ?? "Organization"}
                open={selectedDialogOpen}
                onClose={handleSelectedDialogClose}
            >
                <OrganizationView partialData={selected} />
            </ViewDialogBase>
            {/* Search component */}
            <BaseSearchPage
                title="Search Organizations"
                searchPlaceholder="Search..."
                sortOptions={SORT_OPTIONS}
                defaultSortOption={SORT_OPTIONS[1]}
                query={organizationsQuery}
                listItemFactory={listItemFactory}
                getOptionLabel={(o: any) => o.name}
                onObjectSelect={(selected: Organization) => setSelected(selected)}
                popupButtonText="Invite"
                popupButtonTooltip="Can't find who you're looking for? Invite them😊"
                onPopupButtonClick={handleSurpriseDialogOpen}
            />
        </>
    )
}