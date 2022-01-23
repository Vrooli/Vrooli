import { RoutineSortBy } from "@local/shared";
import { RoutineListItem, RoutineView, ShareDialog, ViewDialogBase } from "components";
import { routinesQuery } from "graphql/query";
import { useCallback, useState } from "react";
import { RoutineDeep as Routine } from "types";
import { labelledSortOptions } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { LabelledSortOption } from "utils";
import { SearchRoutinesPageProps } from "./types";

const SORT_OPTIONS: LabelledSortOption<RoutineSortBy>[] = labelledSortOptions(RoutineSortBy);

export const SearchRoutinesPage = ({
    session
}: SearchRoutinesPageProps) => {
    // Handles dialog when selecting a search result
    const [selected, setSelected] = useState<Routine | undefined>(undefined);
    const selectedDialogOpen = Boolean(selected);
    const handleSelectedDialogClose = useCallback(() => setSelected(undefined), []);

    // Handles dialog for the button that appears after scrolling a certain distance
    const [surpriseDialogOpen, setSurpriseDialogOpen] = useState(false);
    const handleSurpriseDialogOpen = useCallback(() => setSurpriseDialogOpen(true), []);
    const handleSurpriseDialogClose = useCallback(() => setSurpriseDialogOpen(false), []);

    const listItemFactory = (node: Routine, index: number) => (
        <RoutineListItem
            key={`routine-list-item-${index}`}
            session={session}
            data={node}
            isOwn={false}
            onClick={(selected: Routine) => setSelected(selected)}
        />)

    return (
        <>
            {/* Invite link dialog */}
            <ShareDialog onClose={handleSurpriseDialogClose} open={surpriseDialogOpen} />
            {/* Selected dialog */}
            <ViewDialogBase
                title={selected?.title ?? "Routine"}
                open={selectedDialogOpen}
                onClose={handleSelectedDialogClose}
            >
                <RoutineView partialData={selected} />
            </ViewDialogBase>
            {/* Search component */}
            <BaseSearchPage
                title="Search Routines"
                searchPlaceholder="Search..."
                sortOptions={SORT_OPTIONS}
                defaultSortOption={SORT_OPTIONS[1]}
                query={routinesQuery}
                listItemFactory={listItemFactory}
                getOptionLabel={(o: any) => o.name}
                onObjectSelect={(selected: Routine) => setSelected(selected)}
                popupButtonText="Add"
                popupButtonTooltip="Can't find what you're looking for? Create it!😎"
                onPopupButtonClick={handleSurpriseDialogOpen}
            />
        </>
    )
}