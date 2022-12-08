import { ProfileUpdateInput, TagHiddenCreateInput, TagHiddenUpdateInput, UserTranslationCreateInput, UserTranslationUpdateInput } from "graphql/generated/globalTypes";
import { Profile, ProfileTranslation, ShapeWrapper, TagHidden } from "types";
import { hasObjectChanged } from "./objectTools";
import { ResourceListShape } from "./resourceShapes";
import { shapePrim, shapeUpdate, shapeUpdateList } from "./shapeTools";
import { shapeTagCreate, shapeTagUpdate, TagShape } from "./tagShapes";
import { shapeResourceListCreate, shapeResourceListUpdate } from "./resourceShapes";

export type ProfileTranslationShape = Omit<ShapeWrapper<ProfileTranslation>, 'language' | 'bio'> & {
    id: string;
    language: UserTranslationCreateInput['language'];
    bio: UserTranslationCreateInput['bio'];
}

export type ProfileShape = Omit<ShapeWrapper<Profile>, 'emails' | 'pushDevices' | 'wallets' | 'translations' | 'schedules'> & {
    id: string;
    translations?: ProfileTranslationShape[] | null;
    schedules?: UserScheduleShape[] | null;
}

export const shapeProfileTranslationCreate = (item: ProfileTranslationShape): UserTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    bio: item.bio,
})

export const shapeProfileTranslationUpdate = (
    original: ProfileTranslationShape,
    updated: ProfileTranslationShape
): UserTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        ...shapePrim(o, u, 'bio'),
    }), 'id')

export const shapeProfileUpdate = (
    original: ProfileShape,
    updated: ProfileShape
): ProfileUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        ...shapePrim(o, u, 'handle'),
        ...shapePrim(o, u, 'isPrivate'),
        ...shapePrim(o, u, 'isPrivateApis'),
        ...shapePrim(o, u, 'isPrivateApisCreated'),
        ...shapePrim(o, u, 'isPrivateMemberships'),
        ...shapePrim(o, u, 'isPrivateOrganizationsCreated'),
        ...shapePrim(o, u, 'isPrivateProjects'),
        ...shapePrim(o, u, 'isPrivateProjectsCreated'),
        ...shapePrim(o, u, 'isPrivatePullRequests'),
        ...shapePrim(o, u, 'isPrivateQuestionsAnswered'),
        ...shapePrim(o, u, 'isPrivateQuestionsAsked'),
        ...shapePrim(o, u, 'isPrivateQuizzesCreated'),
        ...shapePrim(o, u, 'isPrivateRoles'),
        ...shapePrim(o, u, 'isPrivateRoutines'),
        ...shapePrim(o, u, 'isPrivateRoutinesCreated'),
        ...shapePrim(o, u, 'isPrivateStandards'),
        ...shapePrim(o, u, 'isPrivateStandardsCreated'),
        ...shapePrim(o, u, 'isPrivateStars'),
        ...shapePrim(o, u, 'isPrivateVotes'),
        ...shapePrim(o, u, 'name'),
        ...shapePrim(o, u, 'theme'),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeProfileTranslationCreate, shapeProfileTranslationUpdate, 'id'),
        ...shapeUpdateList(o, u, 'schedules', hasObjectChanged, shapeUserScheduleCreate, shapeUserScheduleUpdate, 'id'),
    }), 'id')