import { bannerImageConfig, profileImageConfig } from "../../utils";
import { user_botCreate, user_botUpdate, user_deleteOne, user_findMany, user_findOne, user_profile, user_profileEmailUpdate, user_profileUpdate } from "../generated";
import { UserEndpoints } from "../logic/user";
import { setupRoutes } from "./base";

const botImagesConfig: UploadConfig<BotCreateInput | BotUpdateInput> = {
    acceptsFiles: true,
    fields: [{
        ...profileImageConfig,
        fieldName: "profileImage",
        fileNameBase: (input) => `${input.id}-profile`,
    }, {
        ...bannerImageConfig,
        fieldName: "bannerImage",
        fileNameBase: (input) => `${input.id}-banner`,
    }],
};

const userImagesConfig: UploadConfig<undefined> = {
    acceptsFiles: true,
    fields: [{
        ...profileImageConfig,
        fieldName: "profileImage",
        fileNameBase: (_, currentUser) => `${currentUser.id}-profile`,
    }, {
        ...bannerImageConfig,
        fieldName: "bannerImage",
        fileNameBase: (_, currentUser) => `${currentUser.id}-banner`,
    }],
};

const icalConfig: UploadConfig<undefined> = {
    acceptsFiles: true,
    fields: [{
        allowedExtensions: ["ics"],
        fieldName: "file",
        fileNameBase: (_, currentUser) => `${currentUser.id}-import`,
        maxFileSize: 1024 * 1024 * 2, // 2MB
    }],
};

export const UserRest = setupRoutes({
    "/bot/:id": {
        put: [UserEndpoints.Mutation.botUpdate, user_botUpdate, botImagesConfig],
    },
    "/bot": {
        post: [UserEndpoints.Mutation.botCreate, user_botCreate, botImagesConfig],
    },
    "/profile": {
        get: [UserEndpoints.Query.profile, user_profile],
        put: [UserEndpoints.Mutation.profileUpdate, user_profileUpdate, userImagesConfig],
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
