import { PrismaType } from "../types";
import { Member, MemberSearchInput, MemberSortBy, MemberUpdateInput } from "../endpoints/types";
import { Displayer, Formatter } from "./types";
import { Prisma } from "@prisma/client";
import { UserModel } from "./user";
import { padSelect } from "../builders";
import { SelectWrap } from "../builders/types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlUpdate: MemberUpdateInput,
    GqlModel: Member,
    GqlSearch: MemberSearchInput,
    GqlSort: MemberSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.memberUpsertArgs['create'],
    PrismaUpdate: Prisma.memberUpsertArgs['update'],
    PrismaModel: Prisma.memberGetPayload<SelectWrap<Prisma.memberSelect>>,
    PrismaSelect: Prisma.memberSelect,
    PrismaWhere: Prisma.memberWhereInput,
}

const __typename = 'Member' as const;

const suppFields = [] as const;
// const formatter = (): Formatter<Model, typeof suppFields> => ({
//     gqlRelMap: {
//         __typename,
//         organization: 'Organization',
//         user: 'User',
//     },
//     prismaRelMap: {
//         __typename,
//         organization: 'Organization',
//         user: 'User',
//         roles: 'Role',
//         invite: 'MeetingInvite',
//     }
// })

const displayer = (): Displayer<Model> => ({
    select: () => ({
        id: true,
        user: padSelect(UserModel.display.select),
    }),
    label: (select, languages) => UserModel.display.label(select.user as any, languages),
})

export const MemberModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.member,
    // TODO needs searcher
    display: displayer(),
    format: {} as any, //formatter(),
})