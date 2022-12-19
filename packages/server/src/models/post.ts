import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Post, PostCreateInput, PostSearchInput, PostSortBy, PostUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: PostCreateInput,
    GqlUpdate: PostUpdateInput,
    GqlModel: Post,
    GqlSearch: PostSearchInput,
    GqlSort: PostSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.postUpsertArgs['create'],
    PrismaUpdate: Prisma.postUpsertArgs['update'],
    PrismaModel: Prisma.postGetPayload<SelectWrap<Prisma.postSelect>>,
    PrismaSelect: Prisma.postSelect,
    PrismaWhere: Prisma.postWhereInput,
}

const __typename = 'Post' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const PostModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.post,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})