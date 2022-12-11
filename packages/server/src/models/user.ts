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

const __typename = 'User' as const;

const suppFields = ['isStarred', 'isViewed'] as const;
const formatter = (): Formatter<User, typeof suppFields> => ({
    relationshipMap: {
        __typename: 'User',
        comments: 'Comment',
        // emails: 'Email',
        // phones: 'Phone',
        projects: 'Project',
        // pushDevices: 'PushDevice',
        starredBy: 'User',
        reportsReceived: 'Report',
        routines: 'Routine',
    },
    joinMap: { starredBy: 'user' },
    countFields: ['reportsCount'],
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ ids, prisma, userData }) => [
            ['isStarred', async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, 'User')],
            ['isViewed', async () => await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, 'User')],
        ],
    },
})

export const searcher = (): Searcher<
    UserSearchInput,
    UserSortBy,
    Prisma.userWhereInput
> => ({
    defaultSort: UserSortBy.StarsDesc,
    sortBy: UserSortBy,
    searchFields: [
        'createdTimeFrame',
        'minStars',
        'minViews',
        'translationLanguages',
        'updatedTimeFrame',
    ],
    searchStringQuery: ({ insensitive, languages }) => ({
        OR: [
            { translations: { some: { language: languages ? { in: languages } : undefined, bio: { ...insensitive } } } },
            { name: { ...insensitive } },
            { handle: { ...insensitive } },
        ]
    }),
})

const validator = (): Validator<
    any,
    any,
    Prisma.userGetPayload<SelectWrap<Prisma.userSelect>>,
    any,
    Prisma.userSelect,
    Prisma.userWhereInput,
    false,
    false
> => ({
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
    permissionResolvers: () => [],
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

const mutater = (): Mutater<
    User,
    false,
    { graphql: ProfileUpdateInput, db: Prisma.userUpsertArgs['update'] },
    false,
    false
> => ({
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

const displayer = (): Displayer<
    Prisma.userSelect,
    Prisma.userGetPayload<SelectWrap<Prisma.userSelect>>
> => ({
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