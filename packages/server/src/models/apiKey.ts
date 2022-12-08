import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { Displayer, GraphQLModelType } from "./types";

const displayer = (): Displayer<
    Prisma.api_keySelect,
    Prisma.api_keyGetPayload<SelectWrap<Prisma.api_keySelect>>
> => ({
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
    delegate: (prisma: PrismaType) => prisma.api_key,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'ApiKey' as GraphQLModelType,
    validate: {} as any,
})