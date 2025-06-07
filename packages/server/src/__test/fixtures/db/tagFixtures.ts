import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Tag model - used for seeding test data
 */

export class TagDbFactory {
    static createMinimal(overrides?: Partial<Prisma.TagCreateInput>): Prisma.TagCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            tag: `tag_${Date.now()}`,
            ...overrides,
        };
    }

    static createWithTranslations(
        translations: Array<{ language: string; description: string }>,
        overrides?: Partial<Prisma.TagCreateInput>
    ): Prisma.TagCreateInput {
        return {
            ...this.createMinimal(overrides),
            translations: {
                create: translations.map(t => ({
                    id: generatePK(),
                    language: t.language,
                    description: t.description,
                })),
            },
        };
    }

    static createPopular(overrides?: Partial<Prisma.TagCreateInput>): Prisma.TagCreateInput {
        return this.createMinimal({
            bookmarks: 100,
            ...overrides,
        });
    }
}

/**
 * Helper to seed tags for testing
 */
export async function seedTags(
    prisma: any,
    tags: string[],
    options?: {
        withTranslations?: boolean;
        popular?: boolean;
    }
) {
    const createdTags = [];

    for (const tag of tags) {
        const tagData = options?.withTranslations
            ? TagDbFactory.createWithTranslations(
                [
                    { language: "en", description: `Description for ${tag}` },
                    { language: "es", description: `Descripción para ${tag}` },
                ],
                { 
                    tag,
                    ...(options?.popular && { bookmarks: Math.floor(Math.random() * 200) + 50 }),
                }
            )
            : TagDbFactory.createMinimal({ 
                tag,
                ...(options?.popular && { bookmarks: Math.floor(Math.random() * 200) + 50 }),
            });

        const created = await prisma.tag.create({ data: tagData });
        createdTags.push(created);
    }

    return createdTags;
}