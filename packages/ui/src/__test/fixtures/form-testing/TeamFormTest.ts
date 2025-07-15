import {
    createTransformFunction,
    DUMMY_ID,
    endpointsTeam,
    teamFormConfig,
    teamTranslationValidation,
    teamValidation,
    type Session,
    type TeamCreateInput,
    type TeamShape,
    type TeamUpdateInput,
} from "@vrooli/shared";
import { createUIFormTestFactory, type UIFormTestConfig } from "./UIFormTestFactory.js";

/**
 * Configuration for Team form testing with data-driven test scenarios
 */
const teamFormTestConfig: UIFormTestConfig<TeamShape, TeamShape, TeamCreateInput, TeamUpdateInput, TeamShape> = {
    // Form metadata
    objectType: "Team",
    formFixtures: {
        minimal: {
            __typename: "Team" as const,
            id: "01234567890123456789",
            bannerImage: null,
            bookmarkLists: [],
            bookmarks: 0,
            canDelete: true,
            canBookmark: true,
            canReact: true,
            canRead: true,
            canUpdate: true,
            created_at: new Date().toISOString(),
            handle: "test_team",
            isOpenToNewMembers: true,
            isPrivate: false,
            profileImage: null,
            reactionSummaries: [],
            tags: [],
            teamSize: 1,
            translations: [{
                __typename: "TeamTranslation" as const,
                id: "01234567890123456780",
                language: "en",
                name: "Test Team",
                bio: "A minimal test team",
            }],
            updated_at: new Date().toISOString(),
            versions: [],
            views: 0,
            you: {
                __typename: "TeamYou" as const,
                canAddMembers: true,
                canDelete: true,
                canBookmark: true,
                canReact: true,
                canRead: true,
                canUpdate: true,
                isBookmarked: false,
                isViewed: false,
                reaction: null,
            },
        },
        complete: {
            __typename: "Team" as const,
            id: "01234567890123456790",
            bannerImage: null,
            bookmarkLists: [],
            bookmarks: 0,
            canDelete: true,
            canBookmark: true,
            canReact: true,
            canRead: true,
            canUpdate: true,
            created_at: new Date().toISOString(),
            handle: "complete_team",
            isOpenToNewMembers: true,
            isPrivate: false,
            profileImage: null,
            reactionSummaries: [],
            tags: [
                { __typename: "Tag" as const, id: "12345678901234567891" },
                { __typename: "Tag" as const, id: "12345678901234567892" },
                { __typename: "Tag" as const, id: "12345678901234567893" },
            ],
            teamSize: 5,
            translations: [{
                __typename: "TeamTranslation" as const,
                id: "01234567890123456781",
                language: "en",
                name: "Complete Test Team",
                bio: "A comprehensive test team with detailed description. This team is focused on testing and quality assurance across multiple domains.",
            }],
            updated_at: new Date().toISOString(),
            versions: [],
            views: 42,
            you: {
                __typename: "TeamYou" as const,
                canAddMembers: true,
                canDelete: true,
                canBookmark: true,
                canReact: true,
                canRead: true,
                canUpdate: true,
                isBookmarked: false,
                isViewed: false,
                reaction: null,
            },
        },
        invalid: {
            __typename: "Team" as const,
            id: "01234567890123456791",
            bannerImage: null,
            bookmarkLists: [],
            bookmarks: 0,
            canDelete: true,
            canBookmark: true,
            canReact: true,
            canRead: true,
            canUpdate: true,
            created_at: new Date().toISOString(),
            handle: "invalid_team",
            isOpenToNewMembers: true,
            isPrivate: false,
            profileImage: null,
            reactionSummaries: [],
            tags: [],
            teamSize: 1,
            translations: [{
                __typename: "TeamTranslation" as const,
                id: "01234567890123456782",
                language: "en",
                name: "", // Invalid: required field is empty
                bio: "",
            }],
            updated_at: new Date().toISOString(),
            versions: [],
            views: 0,
            you: {
                __typename: "TeamYou" as const,
                canAddMembers: true,
                canDelete: true,
                canBookmark: true,
                canReact: true,
                canRead: true,
                canUpdate: true,
                isBookmarked: false,
                isViewed: false,
                reaction: null,
            },
        },
        edgeCase: {
            __typename: "Team" as const,
            id: "01234567890123456792",
            bannerImage: null,
            bookmarkLists: [],
            bookmarks: 0,
            canDelete: true,
            canBookmark: true,
            canReact: true,
            canRead: true,
            canUpdate: true,
            created_at: new Date().toISOString(),
            handle: "edge_case_team",
            isOpenToNewMembers: false,
            isPrivate: true,
            profileImage: null,
            reactionSummaries: [],
            tags: [
                { __typename: "Tag" as const, id: "12345678901234567894" },
                { __typename: "Tag" as const, id: "12345678901234567895" },
                { __typename: "Tag" as const, id: "12345678901234567896" },
                { __typename: "Tag" as const, id: "12345678901234567897" },
            ],
            teamSize: 1,
            translations: [{
                __typename: "TeamTranslation" as const,
                id: "01234567890123456783",
                language: "en",
                name: "A".repeat(200), // Edge case: very long name
                bio: "Bio with special characters: @#$%^&*()[]{}|\\:;\"'<>,.?/~`\n\nMultiple\nline\nbreaks\n\nAnd emoji: 🚀 🎯 ✅",
            }],
            updated_at: new Date().toISOString(),
            versions: [],
            views: 0,
            you: {
                __typename: "TeamYou" as const,
                canAddMembers: true,
                canDelete: true,
                canBookmark: true,
                canReact: true,
                canRead: true,
                canUpdate: true,
                isBookmarked: false,
                isViewed: false,
                reaction: null,
            },
        },
    },

    // Validation schemas from shared package
    validation: teamValidation,
    translationValidation: teamTranslationValidation,

    // API endpoints from shared package
    endpoints: {
        create: endpointsTeam.createOne,
        update: endpointsTeam.updateOne,
    },

    // Transform functions - filter fixture data to only include TeamShape properties
    formToShape: (formData: TeamShape) => {
        // If null or undefined, throw error to catch the issue early
        if (!formData || typeof formData !== 'object') {
            throw new Error('formData must be a non-null object');
        }
        
        // Extract only the properties that TeamShape expects
        const shaped: TeamShape = {
            __typename: "Team" as const,
            id: formData.id || DUMMY_ID,
            config: formData.config || undefined,
            handle: formData.handle || undefined,
            isOpenToNewMembers: formData.isOpenToNewMembers ?? false,
            isPrivate: formData.isPrivate ?? false,
            bannerImage: formData.bannerImage || undefined,
            profileImage: formData.profileImage || undefined,
            tags: formData.tags || [],
            translations: formData.translations || [],
            memberInvites: formData.memberInvites || undefined,
            members: formData.members || undefined,
            membersDelete: formData.membersDelete || undefined,
        };
        return shaped;
    },

    transformFunction: (shape: TeamShape, existing: TeamShape, isCreate: boolean) => {
        // For create operations, existing should be the initial values
        const existingData = isCreate ? teamFormConfig.transformations.getInitialValues() : existing;
        
        const result = createTransformFunction(teamFormConfig)(shape, existingData, isCreate);
        if (!result) {
            throw new Error("Transform function returned undefined");
        }
        return result;
    },

    initialValuesFunction: (session?: Session, existing?: Partial<TeamShape>): TeamShape => {
        return teamFormConfig.transformations.getInitialValues(session, existing);
    },

    // DATA-DRIVEN TEST SCENARIOS - replaces all custom wrapper methods
    testScenarios: {
        nameValidation: {
            description: "Test team name validation and length limits",
            testCases: [
                {
                    name: "Valid team name",
                    field: "translations.0.name",
                    value: "Valid Team Name",
                    shouldPass: true,
                },
                {
                    name: "Empty name",
                    field: "translations.0.name",
                    value: "",
                    shouldPass: false,
                },
                {
                    name: "Very long name",
                    field: "translations.0.name",
                    value: "A".repeat(256),
                    shouldPass: true,
                },
                {
                    name: "Name with special characters",
                    field: "translations.0.name",
                    value: "Team-With-Dashes & Symbols!",
                    shouldPass: true,
                },
            ],
        },

        privacySettings: {
            description: "Test team privacy and membership settings",
            testCases: [
                {
                    name: "Public team, open to new members",
                    data: { isPrivate: false, isOpenToNewMembers: true },
                    shouldPass: true,
                },
                {
                    name: "Public team, closed to new members",
                    data: { isPrivate: false, isOpenToNewMembers: false },
                    shouldPass: true,
                },
                {
                    name: "Private team, closed to new members",
                    data: { isPrivate: true, isOpenToNewMembers: false },
                    shouldPass: true,
                },
            ],
        },

        bioValidation: {
            description: "Test team bio validation",
            testCases: [
                {
                    name: "Valid bio",
                    field: "translations.0.bio",
                    value: "This is a valid team bio",
                    shouldPass: true,
                },
                {
                    name: "Empty bio",
                    field: "translations.0.bio",
                    value: "",
                    shouldPass: true, // Bio is optional
                },
                {
                    name: "Bio with markdown",
                    field: "translations.0.bio",
                    value: "# Team Bio\n\nThis team focuses on **quality** and *testing*.",
                    shouldPass: true,
                },
                {
                    name: "Very long bio",
                    field: "translations.0.bio",
                    value: "A".repeat(2048),
                    shouldPass: true,
                },
            ],
        },

        tagValidation: {
            description: "Test team tag functionality",
            testCases: [
                {
                    name: "No tags",
                    data: { tags: [] },
                    shouldPass: true,
                },
                {
                    name: "Single tag",
                    data: { tags: [{ __typename: "Tag", id: "12345678901234567891" }] },
                    shouldPass: true,
                },
                {
                    name: "Multiple tags",
                    data: {
                        tags: [
                            { __typename: "Tag", id: "12345678901234567891" },
                            { __typename: "Tag", id: "12345678901234567892" },
                            { __typename: "Tag", id: "12345678901234567893" },
                        ],
                    },
                    shouldPass: true,
                },
            ],
        },
    },
};

/**
 * SIMPLIFIED: Direct factory export - no wrapper function needed!
 */
export const teamFormTestFactory = createUIFormTestFactory(teamFormTestConfig);

/**
 * Type exports for use in other test files
 */
export { teamFormTestConfig };
export type { TeamShape as TeamFormData };
