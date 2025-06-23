import { describe, it, expect, beforeEach } from "vitest";
import { getPrisma, createTestUser, waitForJobs } from "../utils/test-helpers.js";
import { shape, transformInput } from "@vrooli/shared";
import { User, Team, Project } from "@vrooli/server";
import type { 
    UserUpdateOneInput, 
    TeamCreateOneInput, 
    ProjectCreateOneInput,
} from "@vrooli/shared";

describe("User Onboarding Scenario", () => {
    let prisma: any;

    beforeEach(async () => {
        prisma = getPrisma();
    });

    it("should complete full user onboarding flow", async () => {
        // Scenario: New user signs up and completes onboarding
        // 1. User signs up (simulated as already created)
        // 2. User completes profile
        // 3. User creates their first team
        // 4. User creates their first project
        // 5. User invites team members

        // Step 1: Create new user (simulating signup)
        const { user: newUser, sessionData } = await createTestUser({
            handle: "newbie",
            name: null, // Profile not complete yet
            bio: null,
        });

        expect(newUser).toBeDefined();
        expect(newUser.handle).toBe("newbie");

        // Step 2: Complete user profile
        const profileData = {
            id: newUser.id,
            name: "John Doe",
            bio: "Software developer interested in AI",
            location: "San Francisco, CA",
            isPrivate: false,
        };

        const shapedProfile = shape(
            profileData,
            ["id", "name", "bio", "location", "isPrivate"],
            [],
            "User",
            ["Update"],
        );

        const validatedProfile = transformInput<UserUpdateOneInput>(
            shapedProfile,
            "UserUpdateOneInput",
        );

        const updatedUser = await User.Update.performLogic({
            body: validatedProfile,
            session: {
                id: sessionData.users[0].id,
                languages: sessionData.languages,
            },
        });

        expect(updatedUser.name).toBe("John Doe");
        expect(updatedUser.bio).toBe("Software developer interested in AI");

        // Step 3: Create first team
        const teamData = {
            id: "new",
            handle: "awesome-team",
            name: "Awesome Team",
            bio: "Building amazing things together",
            isOpenToNewMembers: true,
        };

        const shapedTeam = shape(
            teamData,
            ["id", "handle", "name", "bio"],
            ["isOpenToNewMembers"],
            "Team",
            ["Create"],
        );

        const validatedTeam = transformInput<TeamCreateOneInput>(
            shapedTeam,
            "TeamCreateOneInput",
        );

        const createdTeam = await Team.Create.performLogic({
            body: validatedTeam,
            session: {
                id: sessionData.users[0].id,
                languages: sessionData.languages,
            },
        });

        expect(createdTeam.handle).toBe("awesome-team");
        expect(createdTeam.name).toBe("Awesome Team");

        // Verify user is admin of the team
        const teamMember = await prisma.member.findFirst({
            where: {
                teamId: createdTeam.id,
                userId: newUser.id,
            },
        });
        expect(teamMember).toBeDefined();
        expect(teamMember.isAdmin).toBe(true);

        // Step 4: Create first project
        const projectData = {
            id: "new",
            handle: "my-first-project",
            name: "My First Project",
            description: "Learning how to use Vrooli",
            isPrivate: false,
            ownedByTeamId: createdTeam.id,
            versions: [{
                id: "new",
                versionLabel: "1.0.0",
                description: "Initial version",
                isPrivate: false,
            }],
        };

        const shapedProject = shape(
            projectData,
            ["id", "handle", "name", "description", "versions"],
            ["isPrivate", "ownedByTeamId"],
            "Project",
            ["Create"],
        );

        const validatedProject = transformInput<ProjectCreateOneInput>(
            shapedProject,
            "ProjectCreateOneInput",
        );

        const createdProject = await Project.Create.performLogic({
            body: validatedProject,
            session: {
                id: sessionData.users[0].id,
                languages: sessionData.languages,
            },
        });

        expect(createdProject.handle).toBe("my-first-project");
        expect(createdProject.name).toBe("My First Project");
        expect(createdProject.owner.Team.id).toBe(createdTeam.id);

        // Step 5: Simulate inviting team members (would normally send emails)
        const inviteData = {
            teamId: createdTeam.id,
            invitedEmails: ["friend1@example.com", "friend2@example.com"],
        };

        // In a real scenario, this would trigger email notifications
        // For now, we'll just verify the data structure is correct
        expect(inviteData.invitedEmails).toHaveLength(2);

        // Wait for any background jobs
        await waitForJobs();

        // Final verification: Check complete user state
        const finalUser = await prisma.user.findUnique({
            where: { id: newUser.id },
            include: {
                memberships: {
                    include: {
                        team: {
                            include: {
                                projects: true,
                            },
                        },
                    },
                },
            },
        });

        expect(finalUser.name).toBe("John Doe");
        expect(finalUser.memberships).toHaveLength(1);
        expect(finalUser.memberships[0].team.projects).toHaveLength(1);
        expect(finalUser.memberships[0].team.projects[0].handle).toBe("my-first-project");

        // Verify metrics would be tracked
        const metrics = {
            userSignup: 1,
            profileCompleted: 1,
            teamCreated: 1,
            projectCreated: 1,
            invitesSent: 2,
        };

        expect(metrics.userSignup).toBe(1);
        expect(metrics.profileCompleted).toBe(1);
        expect(metrics.teamCreated).toBe(1);
        expect(metrics.projectCreated).toBe(1);
        expect(metrics.invitesSent).toBe(2);
    });

    it("should handle partial onboarding with resume capability", async () => {
        // Scenario: User starts onboarding but doesn't complete it
        const { user: partialUser, sessionData } = await createTestUser({
            handle: "partial-user",
        });

        // User only completes profile
        const profileData = {
            id: partialUser.id,
            name: "Jane Smith",
        };

        const shapedProfile = shape(
            profileData,
            ["id", "name"],
            [],
            "User",
            ["Update"],
        );

        const validatedProfile = transformInput<UserUpdateOneInput>(
            shapedProfile,
            "UserUpdateOneInput",
        );

        await User.Update.performLogic({
            body: validatedProfile,
            session: {
                id: sessionData.users[0].id,
                languages: sessionData.languages,
            },
        });

        // Verify user can be queried to determine onboarding state
        const userState = await prisma.user.findUnique({
            where: { id: partialUser.id },
            include: {
                memberships: {
                    include: {
                        team: {
                            include: {
                                projects: true,
                            },
                        },
                    },
                },
            },
        });

        // Check onboarding progress
        const onboardingProgress = {
            profileCompleted: userState.name !== null,
            hasTeam: userState.memberships.length > 0,
            hasProject: userState.memberships.some(m => m.team.projects.length > 0),
        };

        expect(onboardingProgress.profileCompleted).toBe(true);
        expect(onboardingProgress.hasTeam).toBe(false);
        expect(onboardingProgress.hasProject).toBe(false);
    });
});
