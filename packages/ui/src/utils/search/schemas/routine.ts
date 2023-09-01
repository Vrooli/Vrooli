import { endpointGetRoutine, endpointGetRoutines, RoutineSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const routineSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchRoutine"),
    containers: [
        hasCompleteVersionContainer,
        votesContainer(),
        bookmarksContainer(),
        languagesVersionContainer(),
        tagsContainer(),
    ],
    fields: [
        ...hasCompleteVersionFields(),
        ...votesFields(),
        ...bookmarksFields(),
        ...languagesVersionFields(),
        ...tagsFields(),
    ],
});

export const routineSearchParams = () => toParams(routineSearchSchema(), endpointGetRoutines, endpointGetRoutine, RoutineSortBy, RoutineSortBy.ScoreDesc);
