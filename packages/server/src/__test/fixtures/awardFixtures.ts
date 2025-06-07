import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Award model - used for seeding test data
 */

export class AwardDbFactory {
    static createMinimal(
        userId: string,
        category: string,
        overrides?: Partial<Prisma.AwardCreateInput>
    ): Prisma.AwardCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            user: { connect: { id: userId } },
            category,
            progress: 0,
            ...overrides,
        };
    }

    static createCompleted(
        userId: string,
        category: string,
        timeCompleted: Date,
        overrides?: Partial<Prisma.AwardCreateInput>
    ): Prisma.AwardCreateInput {
        return this.createMinimal(userId, category, {
            progress: 100,
            timeCompleted,
            ...overrides,
        });
    }

    static createInProgress(
        userId: string,
        category: string,
        progress: number,
        overrides?: Partial<Prisma.AwardCreateInput>
    ): Prisma.AwardCreateInput {
        return this.createMinimal(userId, category, {
            progress,
            ...overrides,
        });
    }
}

/**
 * Helper to seed awards for testing
 */
export async function seedAwards(
    prisma: any,
    options: {
        userId: string;
        categories: Array<{ name: string; progress?: number; completed?: boolean }>;
    }
) {
    const awards = [];

    for (const cat of options.categories) {
        let awardData: Prisma.AwardCreateInput;

        if (cat.completed) {
            awardData = AwardDbFactory.createCompleted(
                options.userId,
                cat.name,
                new Date()
            );
        } else if (cat.progress !== undefined) {
            awardData = AwardDbFactory.createInProgress(
                options.userId,
                cat.name,
                cat.progress
            );
        } else {
            awardData = AwardDbFactory.createMinimal(
                options.userId,
                cat.name
            );
        }

        const award = await prisma.award.create({ data: awardData });
        awards.push(award);
    }

    return awards;
}