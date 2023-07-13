import { user_botCreate, user_botUpdate, user_deleteOne, user_findMany, user_findOne, user_profile, user_profileEmailUpdate, user_profileUpdate } from "../generated";
import { UserEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const UserRest = setupRoutes({
    "/bot/:id": {
        put: [UserEndpoints.Mutation.botUpdate, user_botUpdate, { acceptsFiles: true }],
    },
    "/bot": {
        post: [UserEndpoints.Mutation.botCreate, user_botCreate, { acceptsFiles: true }],
    },
    "/profile": {
        get: [UserEndpoints.Query.profile, user_profile],
        put: [UserEndpoints.Mutation.profileUpdate, user_profileUpdate, { acceptsFiles: true }],
    },
    "/user/:id": {
        get: [UserEndpoints.Query.user, user_findOne],
    },
    "/users": {
        get: [UserEndpoints.Query.users, user_findMany],
    },
    "/profile/email": {
        put: [UserEndpoints.Mutation.profileEmailUpdate, user_profileEmailUpdate],
    },
    "/user": {
        delete: [UserEndpoints.Mutation.userDeleteOne, user_deleteOne],
    },
    // "/importCalendar": {
    //     post: [UserEndpoints.Mutation.importCalendar, importCalendar, { acceptsFiles: true }],
    // },
    // "/exportCalendar": {
    //     get: [UserEndpoints.Mutation.exportCalendar, exportCalendar],
    // },
    // "/exportData": {
    //     get: [UserEndpoints.Mutation.exportData, exportData],
    // },
});
