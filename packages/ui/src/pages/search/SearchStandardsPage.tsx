import { StandardListItem, StandardView, ShareDialog, ViewDialogBase, StandardSortOptions, standardDefaultSortOption, standardOptionLabel } from "components";
import { standardsQuery } from "graphql/query";
import { useCallback, useMemo, useState } from "react";
import { Standard } from "types";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchStandardsPageProps } from "./types";
import { useLocation, useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";

export const SearchStandardsPage = ({
    session
}: SearchStandardsPageProps) => {
    const [, setLocation] = useLocation();
    const [match, params] = useRoute(`${APP_LINKS.SearchStandards}/:id`);
    // Handles dialog when selecting a search result
    const [selected, setSelected] = useState<Standard | undefined>(undefined);
    const selectedDialogOpen = Boolean(match || selected);
    const handleSelected = useCallback((selected: Standard) => {
        setSelected(selected);
        setLocation(`${APP_LINKS.SearchStandards}/${selected.id}`);
    }, [setLocation]);
    const handleSelectedDialogClose = useCallback(() => {
        setSelected(undefined);
        // If selected data exists, then we know we can go back to the previous page
        if (selected) window.history.back();
        // Otherwise the user must have entered the page directly, so we can navigate to the search page
        else setLocation(APP_LINKS.SearchStandards);
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

    const listItemFactory = (node: Standard, index: number) => (
        <StandardListItem
            key={`standard-list-item-${index}`}
            session={session}
            data={node}
            isOwn={false}
            onClick={(selected: Standard) => setSelected(selected)}
        />)

    return (
        <>
            {/* Invite link dialog */}
            <ShareDialog onClose={handleSurpriseDialogClose} open={surpriseDialogOpen} />
            {/* Selected dialog */}
            <ViewDialogBase
                title='View Standard'
                open={selectedDialogOpen}
                onClose={handleSelectedDialogClose}
            >
                <StandardView session={session} partialData={partialData} />
            </ViewDialogBase>
            {/* Search component */}
            <BaseSearchPage
                title="Standards"
                searchPlaceholder="Search..."
                sortOptions={StandardSortOptions}
                defaultSortOption={standardDefaultSortOption}
                query={standardsQuery}
                listItemFactory={listItemFactory}
                getOptionLabel={(o: any) => o.name}
                onObjectSelect={standardOptionLabel}
                showAddButton={true}
                onAddClick={() => {}}
                popupButtonText="Add"
                popupButtonTooltip="Can't find what you're looking for? Create it!😎"
                onPopupButtonClick={handleSurpriseDialogOpen}
            />
        </>
    )
}