import { ProfileUpdateInput, User, UserTranslation, UserTranslationCreateInput, UserTranslationUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { FocusModeShape, shapeFocusMode } from "./focusMode";
import { createPrims, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type ProfileTranslationShape = Pick<UserTranslation, "id" | "language" | "bio"> & {
    __typename?: "UserTranslation";
}

export type ProfileShape = Partial<Pick<User, "handle" | "isPrivate" | "isPrivateApis" | "isPrivateApisCreated" | "isPrivateMemberships" | "isPrivateTeamsCreated" | "isPrivateProjects" | "isPrivateProjectsCreated" | "isPrivatePullRequests" | "isPrivateQuestionsAnswered" | "isPrivateQuestionsAsked" | "isPrivateQuizzesCreated" | "isPrivateRoles" | "isPrivateRoutines" | "isPrivateRoutinesCreated" | "isPrivateStandards" | "isPrivateStandardsCreated" | "isPrivateBookmarks" | "isPrivateVotes" | "name" | "theme">> & {
    __typename: "User",
    id: string;
    bannerImage?: string | File | null;
    focusModes?: FocusModeShape[] | null;
    profileImage?: string | File | null;
    translations?: ProfileTranslationShape[] | null;
}

export const shapeProfileTranslation: ShapeModel<ProfileTranslationShape, UserTranslationCreateInput, UserTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "bio"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "bio"), a),
};

export const shapeProfile: ShapeModel<ProfileShape, null, ProfileUpdateInput> = {
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, null,
            "bannerImage",
            "handle",
            "isPrivate",
            "isPrivateApis",
            "isPrivateApisCreated",
            "isPrivateMemberships",
            "isPrivateProjects",
            "isPrivateProjectsCreated",
            "isPrivatePullRequests",
            "isPrivateQuestionsAnswered",
            "isPrivateQuestionsAsked",
            "isPrivateQuizzesCreated",
            "isPrivateRoles",
            "isPrivateRoutines",
            "isPrivateRoutinesCreated",
            "isPrivateStandards",
            "isPrivateStandardsCreated",
            "isPrivateTeamsCreated",
            "isPrivateBookmarks",
            "isPrivateVotes",
            "name",
            "profileImage",
            "theme",
        ),
        ...updateRel(o, u, "focusModes", ["Create", "Update", "Delete"], "many", shapeFocusMode),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeProfileTranslation),
    }, a),
};
