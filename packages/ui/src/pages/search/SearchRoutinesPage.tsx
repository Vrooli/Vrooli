import { APP_LINKS, RoutineSortBy } from "@local/shared";
import { projectDefaultSortOption, projectOptionLabel, ProjectSortOptions, RoutineListItem, RoutineView, ShareDialog, ViewDialogBase } from "components";
import { routinesQuery } from "graphql/query";
import { useCallback, useMemo, useState } from "react";
import { RoutineDeep as Routine, RoutineDeep } from "types";
import { labelledSortOptions } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { LabelledSortOption } from "utils";
import { SearchRoutinesPageProps } from "./types";
import { useLocation, useRoute } from "wouter";

export const SearchRoutinesPage = ({
    session
}: SearchRoutinesPageProps) => {
    const [, setLocation] = useLocation();
    const [match, params] = useRoute(`${APP_LINKS.SearchRoutines}/:id`);
    // Handles dialog when selecting a search result
    const [selected, setSelected] = useState<RoutineDeep | undefined>(undefined);
    const selectedDialogOpen = Boolean(match || selected);
    const handleSelected = useCallback((selected: RoutineDeep) => {
        setSelected(selected);
        setLocation(`${APP_LINKS.SearchRoutines}/${selected.id}`);
    }, [setLocation]);
    const handleSelectedDialogClose = useCallback(() => {
        setSelected(undefined);
        // If selected data exists, then we know we can go back to the previous page
        if (selected) window.history.back();
        // Otherwise the user must have entered the page directly, so we can navigate to the search page
        else setLocation(APP_LINKS.SearchRoutines);
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
                title='View Routine'
                open={selectedDialogOpen}
                onClose={handleSelectedDialogClose}
            >
                <RoutineView session={session} partialData={partialData} />
            </ViewDialogBase>
            {/* Search component */}
            <BaseSearchPage
                title="Routines"
                searchPlaceholder="Search..."
                sortOptions={ProjectSortOptions}
                defaultSortOption={projectDefaultSortOption}
                query={routinesQuery}
                listItemFactory={listItemFactory}
                getOptionLabel={(o: any) => o.name}
                onObjectSelect={projectOptionLabel}
                showAddButton={true}
                onAddClick={() => {}}
                popupButtonText="Add"
                popupButtonTooltip="Can't find what you're looking for? Create it!😎"
                onPopupButtonClick={handleSurpriseDialogOpen}
            />
        </>
    )
}