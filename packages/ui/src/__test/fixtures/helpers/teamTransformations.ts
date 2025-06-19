import { type TeamCreateInput, type Team, generatePK } from "@vrooli/shared";

/**
 * Transformation utilities for converting between different team data formats
 * These functions handle the data flow between UI forms, API requests, and responses
 */

/**
 * Mock API service functions for testing
 * These simulate the actual API calls that would be made
 */
export const mockTeamService = {
    /**
     * Simulate creating a team via API
     */
    async create(request: TeamCreateInput): Promise<Team> {
        // Simulate API delay
        await new Promise(resolve => setTimeout(resolve, 100));
        
        // Create the team response
        const primaryTranslation = request.translationsCreate?.[0];
        const teamName = primaryTranslation?.name || "Test Team";
        
        const team: Team = {
            __typename: "Team",
            id: request.id,
            handle: request.handle,
            name: teamName,
            isPrivate: request.isPrivate || false,
            isOpenToNewMembers: request.isOpenToNewMembers || false,
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            bannerImage: typeof request.bannerImage === 'string' ? request.bannerImage : null,
            profileImage: typeof request.profileImage === 'string' ? request.profileImage : null,
            membersCount: 1, // Creator is always the first member
            permissions: JSON.stringify(["Read", "Write", "Admin"]),
            translations: request.translationsCreate?.map(trans => ({
                __typename: "TeamTranslation" as const,
                id: trans.id,
                language: trans.language,
                name: trans.name,
                bio: trans.bio || "",
            })) || [
                {
                    __typename: "TeamTranslation" as const,
                    id: "123456789012345679",
                    language: "en",
                    name: teamName,
                    bio: "",
                }
            ],
            you: {
                __typename: "TeamYou",
                canDelete: true,
                canRead: true,
                canUpdate: true,
                canBookmark: false, // Can't bookmark own team
                canReact: false,
                canReport: false,
                isBookmarked: false,
                reaction: null,
                isReported: false,
            },
        };

        // Store in global test storage for retrieval by findById
        const storage = (globalThis as any).__testTeamStorage || {};
        storage[request.id] = JSON.parse(JSON.stringify(team)); // Store a deep copy
        (globalThis as any).__testTeamStorage = storage;
        
        return team;
    },

    /**
     * Simulate fetching a team by ID
     * In a real implementation, this would fetch from database by ID
     */
    async findById(id: string): Promise<Team> {
        await new Promise(resolve => setTimeout(resolve, 50));
        
        // For testing, we'll return the same team that was created
        const storedTeams = (globalThis as any).__testTeamStorage || {};
        if (storedTeams[id]) {
            // Return a deep copy to prevent mutations
            return JSON.parse(JSON.stringify(storedTeams[id]));
        }
        
        // Fallback for testing - create a minimal response
        return {
            __typename: "Team",
            id,
            handle: "fallback-team",
            name: "Fallback Team",
            isPrivate: false,
            isOpenToNewMembers: false,
            createdAt: "2024-01-01T00:00:00Z",
            updatedAt: "2024-01-01T00:00:00Z",
            bannerImage: null,
            profileImage: null,
            membersCount: 1,
            permissions: JSON.stringify(["Read"]),
            translations: [
                {
                    __typename: "TeamTranslation",
                    id: "123456789012345679",
                    language: "en",
                    name: "Fallback Team",
                    bio: "",
                }
            ],
            you: {
                __typename: "TeamYou",
                canDelete: true,
                canRead: true,
                canUpdate: true,
                canBookmark: false,
                canReact: false,
                canReport: false,
                isBookmarked: false,
                reaction: null,
                isReported: false,
            },
        };
    },

    /**
     * Simulate updating a team
     */
    async update(id: string, updates: any): Promise<Team> {
        await new Promise(resolve => setTimeout(resolve, 75));
        
        const team = await this.findById(id);
        
        // Create a deep copy to avoid mutating the original object
        const updatedTeam = JSON.parse(JSON.stringify(team));
        
        // Apply updates
        if (updates.handle !== undefined) updatedTeam.handle = updates.handle;
        if (updates.isPrivate !== undefined) updatedTeam.isPrivate = updates.isPrivate;
        if (updates.isOpenToNewMembers !== undefined) updatedTeam.isOpenToNewMembers = updates.isOpenToNewMembers;
        if (updates.bannerImage !== undefined) {
            updatedTeam.bannerImage = typeof updates.bannerImage === 'string' ? updates.bannerImage : null;
        }
        if (updates.profileImage !== undefined) {
            updatedTeam.profileImage = typeof updates.profileImage === 'string' ? updates.profileImage : null;
        }
        
        // Handle translation updates
        if (updates.translationsCreate) {
            const newTranslations = updates.translationsCreate.map((trans: any) => ({
                __typename: "TeamTranslation" as const,
                id: trans.id,
                language: trans.language,
                name: trans.name,
                bio: trans.bio,
            }));
            updatedTeam.translations = [...(updatedTeam.translations || []), ...newTranslations];
            
            // Update the team name from the primary translation if it exists
            const primaryTranslation = newTranslations.find((t: any) => t.language === 'en') || newTranslations[0];
            if (primaryTranslation) {
                updatedTeam.name = primaryTranslation.name;
            }
        }
        
        updatedTeam.updatedAt = new Date().toISOString();
        
        // Update in storage
        const storage = (globalThis as any).__testTeamStorage || {};
        storage[id] = updatedTeam;
        (globalThis as any).__testTeamStorage = storage;
        
        return updatedTeam;
    },

    /**
     * Simulate deleting a team
     */
    async delete(id: string): Promise<{ success: boolean }> {
        await new Promise(resolve => setTimeout(resolve, 25));
        
        // Remove from storage
        const storage = (globalThis as any).__testTeamStorage || {};
        delete storage[id];
        (globalThis as any).__testTeamStorage = storage;
        
        return { success: true };
    },
};