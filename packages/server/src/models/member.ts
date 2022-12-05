import { PrismaType } from "../types";
import { Member } from "../endpoints/types";
import { Displayer, Formatter, GraphQLModelType } from "./types";
import { Prisma } from "@prisma/client";
import { UserModel } from "./user";
import { padSelect } from "../builders";

const formatter = (): Formatter<Member, any> => ({
    relationshipMap: {
        __typename: 'Member',
        organization: 'Organization',
        user: 'User',
    }
})

const displayer = (): Displayer<
    Prisma.memberSelect,
    Prisma.memberGetPayload<{ select: { [K in keyof Required<Prisma.memberSelect>]: true } }>
> => ({
    select: () => ({
        id: true,
        user: padSelect(UserModel.display.select),
    }),
    label: (select, languages) => UserModel.display.label(select.user as any, languages),
})

export const MemberModel = ({
    delegate: (prisma: PrismaType) => prisma.member,
    // TODO needs searcher
    display: displayer(),
    format: formatter(),
    type: 'Member' as GraphQLModelType,
})