import { BotCreateInput, BotUpdateInput, User } from "@local/shared";
import { ShapeModel } from "types";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
import { shapeUserTranslation } from "./user";

/** Translation for bot-specific properties (which are stringified and stored in `isBot` field) */
export type BotTranslationShape = {
    id: string;
    language: string;
    bias?: string | null;
    domainKnowledge?: string | null;
    keyPhrases?: string | null;
    occupation?: string | null;
    persona?: string | null;
    tone?: string | null;
}

export type BotShape = Pick<User, "id" | "isPrivate" | "name"> & {
    __typename?: "User";
    creativity?: number | null;
    isBot?: true;
    translations?: BotTranslationShape[] | null;
    verbosity?: number | null;
}

export const shapeBotTranslation: ShapeModel<BotTranslationShape, Record<string, string | number>, Record<string, string | number>> = {
    create: (d) => createPrims(d, "bias", "domainKnowledge", "keyPhrases", "occupation", "persona", "tone"),
    /** 
     * Unlike typical updates, we want to include every field so that 
     * we can stringify the entire object and store it in the `botSettings` field. 
     * This means we'll use `createPrims` again.
     **/
    update: (_, u) => createPrims(u, "bias", "domainKnowledge", "keyPhrases", "occupation", "persona", "tone"),
};

export const shapeBot: ShapeModel<BotShape, BotCreateInput, BotUpdateInput> = {
    create: (d) => {
        // Extract bot settings from translations
        const textData = createRel(d, "translations", ["Create"], "many", shapeBotTranslation);
        // Convert to object, where keys are language codes and values are the bot settings
        const textSettings = Object.fromEntries(textData.translationsCreate.map(t => [t.language, t]));
        return {
            isBot: true,
            botSettings: JSON.stringify({
                translations: textSettings,
                creativity: d.creativity ?? undefined,
                verbosity: d.verbosity ?? undefined,
            }),
            ...createPrims(d, "id", "isBot", "name"),
            ...createRel(d, "translations", ["Create"], "many", shapeUserTranslation),
        };
    },
    update: (o, u, a) => {
        // Extract bot settings from translations
        const textData = updateRel({ ...o, translations: [] }, u, "translations", ["Create", "Update", "Delete"], "many", shapeBotTranslation); // Use empty array for original translations to ensure that all translations are included
        // Convert created to object.
        // Note: Shouldn't have to worry about updated since we set the original to empty array (making all translations seem like creates).
        const textSettings = Object.fromEntries(textData.translationsCreate.map(t => [t.language, t]));
        // Since we set the original to empty array, we need to manually remove the deleted translations (i.e. translations in the original but not in the update)
        const deletedTranslations = o.translations?.filter(t => !u.translations?.some(t2 => t2.id === t.id));
        if (deletedTranslations) {
            deletedTranslations.forEach(t => delete textSettings[t.language]);
        }
        return shapeUpdate(u, {
            botSettings: JSON.stringify({
                translations: textSettings,
                creativity: u.creativity ?? undefined,
                verbosity: u.verbosity ?? undefined,
            }),
            ...updatePrims(o, u, "id", "isBot", "name"),
            ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeUserTranslation),
        }, a);
    },
};
