/**
 * Team Database Factory
 * 
 * Creates team records in the test database
 */

import { generatePK, generatePublicId } from "@vrooli/shared";
import { DbProvider } from "../../../../db/provider.js";
import type { TeamTestData } from "../../types.js";

export class TeamDbFactory {
    async create(teamData: TeamTestData): Promise<any> {
        const team = await DbProvider.get().team.create({
            data: {
                id: BigInt(teamData.id),
                publicId: generatePublicId(),
                isPrivate: false,
                // Skip user relations for now to avoid foreign key issues
                translations: {
                    create: {
                        id: generatePK(),
                        language: "en",
                        name: teamData.name,
                        bio: teamData.details || teamData.businessPrompt,
                    },
                },
            },
            include: {
                translations: true,
            },
        });

        return team;
    }

    async createBatch(teams: TeamTestData[]): Promise<any[]> {
        return Promise.all(teams.map(t => this.create(t)));
    }

    async update(id: string, data: Partial<TeamTestData>): Promise<any> {
        return DbProvider.get().team.update({
            where: { id: BigInt(id) },
            data: {
                translations: data.name || data.description ? {
                    update: {
                        where: { language: "en" },
                        data: {
                            name: data.name,
                            bio: data.description,
                        },
                    },
                } : undefined,
            },
        });
    }

    async delete(id: string): Promise<void> {
        await DbProvider.get().team.delete({
            where: { id: BigInt(id) },
        });
    }
}
