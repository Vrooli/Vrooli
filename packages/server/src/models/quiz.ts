import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MaxObjects, Quiz, QuizCreateInput, QuizSearchInput, QuizSortBy, QuizUpdateInput, QuizYou } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel, defaultPermissions, oneIsPublic } from "../utils";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";
import { BookmarkModel } from "./bookmark";
import { VoteModel } from "./vote";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { quizValidation } from "@shared/validation";

const __typename = 'Quiz' as const;
type Permissions = Pick<QuizYou, 'canDelete' | 'canUpdate' | 'canBookmark' | 'canRead' | 'canVote'>;
const suppFields = ['you'] as const;
export const QuizModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: QuizCreateInput,
    GqlUpdate: QuizUpdateInput,
    GqlModel: Quiz,
    GqlSearch: QuizSearchInput,
    GqlSort: QuizSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.quizUpsertArgs['create'],
    PrismaUpdate: Prisma.quizUpsertArgs['update'],
    PrismaModel: Prisma.quizGetPayload<SelectWrap<Prisma.quizSelect>>,
    PrismaSelect: Prisma.quizSelect,
    PrismaWhere: Prisma.quizWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.quiz,
    display: {
        select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages)
    },
    format: {
        gqlRelMap: {
            __typename,
            attempts: 'QuizAttempt',
            createdBy: 'User',
            project: 'Project',
            quizQuestions: 'QuizQuestion',
            routine: 'Routine',
            bookmarkedBy: 'User',
        },
        prismaRelMap: {
            __typename,
            attempts: 'QuizAttempt',
            createdBy: 'User',
            project: 'Project',
            quizQuestions: 'QuizQuestion',
            routine: 'Routine',
            bookmarkedBy: 'User',
        },
        joinMap: { bookmarkedBy: 'user' },
        countFields: {
            attemptsCount: true,
            quizQuestionsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        hasCompleted: new Array(ids.length).fill(false), // TODO: Implement
                        isBookmarked: await BookmarkModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        isUpvoted: await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename),
                    }
                }
            },
        },
    },
    mutate: {
        shape: {
            create: async ({ data, prisma, userData }) => ({
                id: data.id,
                //TODO
            } as any),
            update: async ({ data, prisma, userData }) => ({
                id: data.id,
                //TODO
            } as any)
        },
        yup: quizValidation,
    },
    search: {
        defaultSort: QuizSortBy.ScoreDesc,
        sortBy: QuizSortBy,
        searchFields: {
            createdTimeFrame: true,
            isComplete: true,
            translationLanguages: true,
            maxBookmarks: true,
            maxScore: true,
            minBookmarks: true,
            minScore: true,
            routineId: true,
            projectId: true,
            userId: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                'transDescriptionWrapped',
                'transNameWrapped',
            ]
        }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => data.isPrivate === false && oneIsPublic<Prisma.quizSelect>(data, [
            ['project', 'Project'],
            ['routine', 'Routine'],
        ], languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data.createdBy,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            isPrivate: true,
            createdBy: 'User',
            project: 'Project',
            routine: 'Routine',
        }),
        visibility: {
            private: {
                OR: [
                    { isPrivate: true },
                    { project: ProjectModel.validate!.visibility.private },
                    { routine: RoutineModel.validate!.visibility.private },
                ]
            },
            public: {
                AND: [
                    { isPrivate: false },
                    { project: ProjectModel.validate!.visibility.public },
                    { routine: RoutineModel.validate!.visibility.public },
                ]
            },
            owner: (userId) => ({ createdBy: { id: userId } }),
        }
    },
})