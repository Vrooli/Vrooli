import { user_botCreate, user_botUpdate, user_deleteOne, user_findMany, user_findOne, user_profile, user_profileEmailUpdate, user_profileUpdate } from "../generated";
import { UserEndpoints } from "../logic";
import { UploadConfig, setupRoutes } from "./base";

const imagesConfig: UploadConfig = {
    acceptsFiles: true,
    imageSizes: [
        { width: 1024, height: 1024 },
        { width: 512, height: 512 },
        { width: 256, height: 256 },
        { width: 128, height: 128 },
        { width: 64, height: 64 },
    ],
};

export const UserRest = setupRoutes({
    "/bot/:id": {
        put: [UserEndpoints.Mutation.botUpdate, user_botUpdate, imagesConfig],
    },
    "/bot": {
        post: [UserEndpoints.Mutation.botCreate, user_botCreate, imagesConfig],
    },
    "/profile": {
        get: [UserEndpoints.Query.profile, user_profile],
        put: [UserEndpoints.Mutation.profileUpdate, user_profileUpdate, imagesConfig],
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
