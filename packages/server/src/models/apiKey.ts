import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { ApiKey } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter } from "./types";

type Model = {
    GqlModel: ApiKey,
    PrismaModel: Prisma.api_keyGetPayload<SelectWrap<Prisma.api_keySelect>>,
    PrismaSelect: Prisma.api_keySelect,
}

const __typename = 'ApiKey' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    relationshipMap: {
        __typename,
    },
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, key: true }),
    // Label should be first 4 characters of key, an ellipsis, and last 4 characters of key
    label: (select) => {
        // Make sure key is at least 8 characters long
        // (should always be, but you never know)
        if (select.key.length < 8) return select.key
        return select.key.slice(0, 4) + '...' + select.key.slice(-4)
    }
})

export const ApiKeyModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.api_key,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    validate: {} as any,
})