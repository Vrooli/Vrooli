import { focusMode_create, focusMode_findMany, focusMode_findOne, focusMode_setActive, focusMode_update } from "@local/shared";
import { FocusModeEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const FocusModeRest = setupRoutes({
    "/focusMode/:id": {
        get: [FocusModeEndpoints.Query.focusMode, focusMode_findOne],
        put: [FocusModeEndpoints.Mutation.focusModeUpdate, focusMode_update],
    },
    "/focusModes": {
        get: [FocusModeEndpoints.Query.focusModes, focusMode_findMany],
    },
    "/focusMode": {
        post: [FocusModeEndpoints.Mutation.focusModeCreate, focusMode_create],
    },
    "/focusMode/active": {
        put: [FocusModeEndpoints.Mutation.setActiveFocusMode, focusMode_setActive],
    },
});
