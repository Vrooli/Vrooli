import { ResourceListShape } from "./resource";
import { shapeTagCreate, shapeTagUpdate, TagShape } from "./tag";
import { shapeResourceListCreate, shapeResourceListUpdate } from "./resource";
import { ProfileUpdateInput, User, User, UserTranslation, UserTranslationCreateInput } from "@shared/consts";
import { createPrims, hasObjectChanged, shapeUpdate, updatePrims } from "utils";
import { ShapeModel } from "types";

export type ProfileTranslationShape = Pick<UserTranslation, 'id' | 'language' | 'bio'>

export type ProfileShape = Omit<OmitCalculated<User>, 'emails' | 'pushDevices' | 'wallets' | 'translations' | 'schedules'> & {
    id: string;
    translations?: ProfileTranslationShape[] | null;
    schedules?: UserScheduleShape[] | null;
}

export const shapeProfileTranslation: ShapeModel<ProfileTranslationShape, UserTranslationCreateInput, UserTranslationUpdateInput> = {
    create: (item) => createPrims(item, 'id', 'language', 'bio'),
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
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeProfileTranslationCreate, shapeProfileTranslationUpdate, 'id'),
        ...shapeUpdateList(o, u, 'schedules', hasObjectChanged, shapeUserScheduleCreate, shapeUserScheduleUpdate, 'id'),
    })