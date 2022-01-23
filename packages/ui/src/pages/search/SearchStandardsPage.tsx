import { StandardSortBy } from "@local/shared";
import { StandardListItem, StandardView, ShareDialog, ViewDialogBase } from "components";
import { standardsQuery } from "graphql/query";
import { useCallback, useState } from "react";
import { Standard } from "types";
import { labelledSortOptions } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { LabelledSortOption } from "utils";
import { SearchStandardsPageProps } from "./types";

const SORT_OPTIONS: LabelledSortOption<StandardSortBy>[] = labelledSortOptions(StandardSortBy);

export const SearchStandardsPage = ({
    session
}: SearchStandardsPageProps) => {
    // Handles dialog when selecting a search result
    const [selected, setSelected] = useState<Standard | undefined>(undefined);
    const selectedDialogOpen = Boolean(selected);
    const handleSelectedDialogClose = useCallback(() => setSelected(undefined), []);

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
                title={selected?.name ?? "Standard"}
                open={selectedDialogOpen}
                onClose={handleSelectedDialogClose}
            >
                <StandardView partialData={selected} />
            </ViewDialogBase>
            {/* Search component */}
            <BaseSearchPage
                title="Search Standards"
                searchPlaceholder="Search..."
                sortOptions={SORT_OPTIONS}
                defaultSortOption={SORT_OPTIONS[1]}
                query={standardsQuery}
                listItemFactory={listItemFactory}
                getOptionLabel={(o: any) => o.name}
                onObjectSelect={(selected: Standard) => setSelected(selected)}
                popupButtonText="Add"
                popupButtonTooltip="Can't find what you're looking for? Create it!😎"
                onPopupButtonClick={handleSurpriseDialogOpen}
            />
        </>
    )
}