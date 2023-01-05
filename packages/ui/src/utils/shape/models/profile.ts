import { ProfileUpdateInput, User, UserTranslation, UserTranslationCreateInput, UserTranslationUpdateInput } from "@shared/consts";
import { createPrims, shapeUpdate, shapeUserSchedule, updatePrims, updateRel, UserScheduleShape } from "utils";
import { ShapeModel } from "types";

export type ProfileTranslationShape = Pick<UserTranslation, 'id' | 'language' | 'bio'>

export type ProfileShape = Pick<User, 'handle' | 'isPrivate' | 'isPrivateApis' | 'isPrivateApisCreated' | 'isPrivateMemberships' | 'isPrivateOrganizationsCreated' | 'isPrivateProjects' | 'isPrivateProjectsCreated' | 'isPrivatePullRequests' | 'isPrivateQuestionsAnswered' | 'isPrivateQuestionsAsked' | 'isPrivateQuizzesCreated' | 'isPrivateRoles' | 'isPrivateRoutines' | 'isPrivateRoutinesCreated' | 'isPrivateStandards' | 'isPrivateStandardsCreated' | 'isPrivateStars' | 'isPrivateVotes' | 'name' | 'theme'> & {
    id: string;
    translations?: ProfileTranslationShape[] | null;
    schedules?: UserScheduleShape[] | null;
}

export const shapeProfileTranslation: ShapeModel<ProfileTranslationShape, UserTranslationCreateInput, UserTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'bio'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'bio'))
}

export const shapeProfileUpdate = (o: ProfileShape, u: ProfileShape): ProfileUpdateInput | undefined =>
    shapeUpdate(u, {
        ...updatePrims(o, u, null, 'handle',
            'isPrivate',
            'isPrivateApis',
            'isPrivateApisCreated',
            'isPrivateMemberships',
            'isPrivateOrganizationsCreated',
            'isPrivateProjects',
            'isPrivateProjectsCreated',
            'isPrivatePullRequests',
            'isPrivateQuestionsAnswered',
            'isPrivateQuestionsAsked',
            'isPrivateQuizzesCreated',
            'isPrivateRoles',
            'isPrivateRoutines',
            'isPrivateRoutinesCreated',
            'isPrivateStandards',
            'isPrivateStandardsCreated',
            'isPrivateStars',
            'isPrivateVotes',
            'name',
            'theme',
        ),
        ...updateRel(o, u, 'schedules', ['Create', 'Update', 'Delete'], 'many', shapeUserSchedule),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeProfileTranslation),
    })