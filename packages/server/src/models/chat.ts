import { PrismaType } from "../types";
import { bestTranslation } from "../utils";
import { ModelLogic } from "./types";

const __typename = "Chat" as const;
const suppFields = ["you"] as const;
export const ChatModel: ModelLogic<any, any> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.chat,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: ({ translations }, languages) => bestTranslation(translations, languages).name ?? "",
        },
        embed: {
            select: () => ({ id: true, translations: { select: { embeddingNeedsUpdate: true, language: true, name: true, description: true } } }),
            get: ({ translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return JSON.stringify({
                    description: trans.description,
                    name: trans.name,
                });
            },
        },
    },
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
});
