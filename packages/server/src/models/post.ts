import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Post, PostCreateInput, PostSearchInput, PostSortBy, PostUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { ModelLogic } from "./types";

const __typename = 'Post' as const;

const suppFields = [] as const;

export const PostModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: PostCreateInput,
    GqlUpdate: PostUpdateInput,
    GqlModel: Post,
    GqlSearch: PostSearchInput,
    GqlSort: PostSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.postUpsertArgs['create'],
    PrismaUpdate: Prisma.postUpsertArgs['update'],
    PrismaModel: Prisma.postGetPayload<SelectWrap<Prisma.postSelect>>,
    PrismaSelect: Prisma.postSelect,
    PrismaWhere: Prisma.postWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.post,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            comments: 'Comment',
            owner: {
                user: 'User',
                organization: 'Organization',
            },
            reports: 'Report',
            repostedFrom: 'Post',
            reposts: 'Post',
            resourceList: 'ResourceList',
            bookmarkedBy: 'User',
            tags: 'Tag',
        },
        prismaRelMap: {
            __typename,
            organization: 'Organization',
            user: 'User',
            repostedFrom: 'Post',
            reposts: 'Post',
            resourceList: 'ResourceList',
            comments: 'Comment',
            bookmarkedBy: 'User',
            votedBy: 'Vote',
            viewedBy: 'View',
            reports: 'Report',
            tags: 'Tag',
        },
        countFields: {
            commentsCount: true,
            repostsCount: true,
        },
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})