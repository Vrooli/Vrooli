import { hasObjectChanged } from "./objectTools";
import { ResourceListShape } from "./resource";
import { shapeCreatePrims, shapeUpdatePrims, shapeUpdate, shapeUpdateList } from "./shapeTools";
import { shapeTagCreate, shapeTagUpdate, TagShape } from "./tag";
import { shapeResourceListCreate, shapeResourceListUpdate } from "./resource";
import { ProfileUpdateInput, User, User, UserTranslation, UserTranslationCreateInput } from "@shared/consts";
import { OmitCalculated } from "types";

export type ProfileTranslationShape = OmitCalculated<UserTranslation>

export type ProfileShape = Omit<OmitCalculated<User>, 'emails' | 'pushDevices' | 'wallets' | 'translations' | 'schedules'> & {
    id: string;
    translations?: ProfileTranslationShape[] | null;
    schedules?: UserScheduleShape[] | null;
}

export const shapeProfileTranslationCreate = (item: ProfileTranslationShape): UserTranslationCreateInput =>
    shapeCreatePrims(item, 'id', 'language', 'bio')

export const shapeProfileTranslationUpdate = (o: ProfileTranslationShape, u: ProfileTranslationShape): UserTranslationUpdateInput | undefined =>
    shapeUpdate(u, shapeUpdatePrims(o, u, 'id', 'bio'))

export const shapeProfileUpdate = (o: ProfileShape, u: ProfileShape): ProfileUpdateInput | undefined =>
    shapeUpdate(u, {
        ...shapeUpdatePrims(o, u, null, 'handle',
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