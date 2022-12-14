import { StarModel } from "./star";
import { ViewModel } from "./view";
import { UserSortBy } from "@shared/consts";
import { ProfileUpdateInput, User, UserSearchInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Formatter, Searcher, Validator, Displayer, Mutater } from "./types";
import { Prisma } from "@prisma/client";
import { profilesUpdate } from "@shared/validation";
import { translationRelationshipBuilder } from "../utils";
import { SelectWrap } from "../builders/types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlUpdate: ProfileUpdateInput,
    GqlModel: User,
    GqlSearch: UserSearchInput,
    GqlSort: UserSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.userUpsertArgs['create'],
    PrismaUpdate: Prisma.userUpsertArgs['update'],
    PrismaModel: Prisma.userGetPayload<SelectWrap<Prisma.userSelect>>,
    PrismaSelect: Prisma.userSelect,
    PrismaWhere: Prisma.userWhereInput,
}

const __typename = 'User' as const;

const suppFields = ['isStarred', 'isViewed'] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    relationshipMap: {
        __typename: 'User',
        comments: 'Comment',
        emails: 'Email',
        // phones: 'Phone',
        projects: 'Project',
        pushDevices: 'PushDevice',
        starredBy: 'User',
        reportsReceived: 'Report',
        routines: 'Routine',
    },
    joinMap: { starredBy: 'user' },
    countFields: ['reportsCount'],
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ ids, prisma, userData }) => [
            ['isStarred', async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename)],
            ['isViewed', async () => await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename)],
        ],
    },
})

export const searcher = (): Searcher<Model> => ({
    defaultSort: UserSortBy.StarsDesc,
    sortBy: UserSortBy,
    searchFields: [
        'createdTimeFrame',
        'minStars',
        'minViews',
        'translationLanguages',
        'updatedTimeFrame',
    ],
    searchStringQuery: () => ({
        OR: [
            'transBioWrapped',
            'nameWrapped',
            'handleWrapped',
        ]
    }),
})

const validator = (): Validator<Model> => ({
    validateMap: {
        __typename: 'User',
        projects: 'Project',
        reportsCreated: 'Report',
        routines: 'Routine',
        // userSchedules: 'UserSchedule',
    },
    isTransferable: false,
    maxObjects: 0,
    permissionsSelect: () => ({
        id: true,
        isPrivate: true,
        languages: { select: { language: true } },
    }),
    permissionResolvers: () => ({}),
    owner: (data) => ({ User: data }),
    isDeleted: () => false,
    isPublic: (data) => data.isPrivate === false,
    profanityFields: ['name', 'handle'],
    visibility: {
        private: { isPrivate: true },
        public: { isPrivate: false },
        owner: (userId) => ({ id: userId }),
    },
    // createMany.forEach(input => lineBreaksCheck(input, ['bio'], 'LineBreaksBio'));
})

const mutater = (): Mutater<Model> => ({
    shape: {
        update: async ({ data, prisma, userData }) => {
            return {
                handle: data.handle,
                name: data.name ?? undefined,
                theme: data.theme ?? undefined,
                // // hiddenTags: await TagHiddenModel.mutate(prisma).relationshipBuilder!(userData.id, input, false),
                // starred: {
                //     create: starredCreate,
                //     delete: starredDelete,
                // }, TODO
                translations: await translationRelationshipBuilder(prisma, userData, data, false),
            }
        }
    },
    yup: { update: profilesUpdate },
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, name: true }),
    label: (select) => select.name ?? '',
})

export const UserModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.user,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    search: searcher(),
    validate: validator(),
})