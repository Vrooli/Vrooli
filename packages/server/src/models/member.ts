import { PrismaType } from "../types";
import { Member } from "../endpoints/types";
import { Displayer, Formatter } from "./types";
import { Prisma } from "@prisma/client";
import { UserModel } from "./user";
import { padSelect } from "../builders";
import { SelectWrap } from "../builders/types";

const __typename = 'Member' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Member, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        organization: 'Organization',
        user: 'User',
    }
})

const displayer = (): Displayer<
    Prisma.memberSelect,
    Prisma.memberGetPayload<SelectWrap<Prisma.memberSelect>>
> => ({
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
    format: formatter(),
})