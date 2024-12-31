import { endpointsFocusMode } from "@local/shared";
import { focusMode_create, focusMode_findMany, focusMode_findOne, focusMode_setActive, focusMode_update } from "../generated";
import { FocusModeEndpoints } from "../logic/focusMode";
import { setupRoutes } from "./base";

export const FocusModeRest = setupRoutes([
    [endpointsFocusMode.findOne, FocusModeEndpoints.Query.focusMode, focusMode_findOne],
    [endpointsFocusMode.findMany, FocusModeEndpoints.Query.focusModes, focusMode_findMany],
    [endpointsFocusMode.createOne, FocusModeEndpoints.Mutation.focusModeCreate, focusMode_create],
    [endpointsFocusMode.updateOne, FocusModeEndpoints.Mutation.focusModeUpdate, focusMode_update],
    [endpointsFocusMode.setActive, FocusModeEndpoints.Mutation.setActiveFocusMode, focusMode_setActive],
]);
