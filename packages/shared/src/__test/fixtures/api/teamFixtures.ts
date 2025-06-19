import { type ModelTestFixtures, TestDataFactory } from "../../../validation/models/__test/validationTestUtils.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
    id6: "123456789012345683",
};

// Shared team test fixtures
export const teamFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            translationsCreate: [
                {
                    id: validIds.id2,
                    language: "en",
                    name: "Minimal Team",
                },
            ],
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            bannerImage: "team-banner.jpg",
            config: { theme: "dark", notifications: true },
            handle: "awesome_team",
            isOpenToNewMembers: true,
            isPrivate: false,
            profileImage: "team-profile.png",
            tagsConnect: [validIds.id3, validIds.id4],
            tagsCreate: [
                {
                    id: validIds.id5,
                    tag: "collaboration",
                },
            ],
            memberInvitesCreate: [
                {
                    id: validIds.id6,
                    message: "Join our amazing team!",
                    teamConnect: validIds.id1,
                    userConnect: validIds.id2,
                },
            ],
            translationsCreate: [
                {
                    id: validIds.id2,
                    language: "en",
                    name: "Awesome Team",
                    bio: "We are building amazing things together.",
                },
                {
                    id: validIds.id3,
                    language: "es",
                    name: "Equipo Increíble",
                    bio: "Estamos construyendo cosas increíbles juntos.",
                },
            ],
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 123,
            unknownField3: true,
        },
        update: {
            id: validIds.id1,
            bannerImage: "new-banner.jpg",
            config: { theme: "light", notifications: false },
            handle: "updated_team",
            isOpenToNewMembers: false,
            isPrivate: true,
            profileImage: "new-profile.png",
            tagsConnect: [validIds.id4],
            tagsDisconnect: [validIds.id3],
            tagsCreate: [
                {
                    id: validIds.id5,
                    tag: "innovation",
                },
            ],
            memberInvitesCreate: [
                {
                    id: validIds.id6,
                    message: "Come join us!",
                    teamConnect: validIds.id1,
                    userConnect: validIds.id3,
                },
            ],
            memberInvitesDelete: [validIds.id2],
            membersDelete: [validIds.id4],
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 456,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required id
                handle: "test_team",
                isOpenToNewMembers: true,
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "Team Name",
                    },
                ],
            },
            update: {
                // Missing required id
                handle: "updated_team",
                isPrivate: false,
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                bannerImage: 456, // Should be string
                config: { invalid: "object" }, // Should be JSON string
                handle: 789, // Should be string
                isOpenToNewMembers: "true", // Should be boolean
                isPrivate: "false", // Should be boolean
                profileImage: 101112, // Should be string
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "Team Name",
                    },
                ],
            },
            update: {
                id: validIds.id1,
                bannerImage: 123, // Should be string
                config: true, // Should be JSON string
                handle: false, // Should be string
                isOpenToNewMembers: "yes", // Should be boolean
                isPrivate: "no", // Should be boolean
                profileImage: 456, // Should be string
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "Invalid ID Team",
                    },
                ],
            },
            update: {
                id: "invalid-id",
            },
        },
        invalidHandle: {
            create: {
                id: validIds.id1,
                handle: "ab", // Too short (min 3)
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "Short Handle Team",
                    },
                ],
            },
        },
        longHandle: {
            create: {
                id: validIds.id1,
                handle: "this_handle_is_way_too_long", // Too long (max 16)
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "Long Handle Team",
                    },
                ],
            },
        },
        invalidHandleChars: {
            create: {
                id: validIds.id1,
                handle: "team-name!", // Invalid characters
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "Invalid Chars Team",
                    },
                ],
            },
        },
        invalidTranslations: {
            create: {
                id: validIds.id1,
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "AB", // Too short (min 3 for name)
                    },
                ],
            },
        },
        missingTranslationName: {
            create: {
                id: validIds.id1,
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        // Missing required name
                        bio: "Team without name",
                    },
                ],
            },
        },
    },
    edgeCases: {
        withoutOptionalFields: {
            create: {
                id: validIds.id1,
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "Basic Team",
                    },
                ],
                // No optional fields
            },
        },
        privateTeam: {
            create: {
                id: validIds.id1,
                isPrivate: true,
                isOpenToNewMembers: false,
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "Private Team",
                        bio: "This is a private team.",
                    },
                ],
            },
        },
        publicOpenTeam: {
            create: {
                id: validIds.id1,
                isPrivate: false,
                isOpenToNewMembers: true,
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "Public Open Team",
                        bio: "Everyone is welcome!",
                    },
                ],
            },
        },
        withAllImages: {
            create: {
                id: validIds.id1,
                bannerImage: "beautiful-banner.jpg",
                profileImage: "team-logo.png",
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "Visual Team",
                    },
                ],
            },
        },
        multipleLanguages: {
            create: {
                id: validIds.id1,
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "Global Team",
                        bio: "We work across the globe.",
                    },
                    {
                        id: validIds.id3,
                        language: "fr",
                        name: "Équipe Mondiale",
                        bio: "Nous travaillons à travers le monde.",
                    },
                    {
                        id: validIds.id4,
                        language: "de",
                        name: "Globales Team",
                        bio: "Wir arbeiten rund um den Globus.",
                    },
                ],
            },
        },
        withTags: {
            create: {
                id: validIds.id1,
                tagsConnect: [validIds.id3, validIds.id4, validIds.id5],
                tagsCreate: [
                    {
                        id: validIds.id6,
                        tag: "teamwork",
                    },
                ],
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "Tagged Team",
                    },
                ],
            },
        },
        withMemberInvites: {
            create: {
                id: validIds.id1,
                memberInvitesCreate: [
                    {
                        id: validIds.id3,
                        message: "Join our awesome team!",
                        teamConnect: validIds.id1,
                        userConnect: validIds.id4,
                    },
                    {
                        id: validIds.id5,
                        message: "We need your skills!",
                        teamConnect: validIds.id1,
                        userConnect: validIds.id6,
                    },
                ],
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "Recruiting Team",
                    },
                ],
            },
        },
        complexConfig: {
            create: {
                id: validIds.id1,
                config: {
                    theme: "dark",
                    notifications: {
                        email: true,
                        push: false,
                        sms: true,
                    },
                    features: ["collaboration", "analytics", "reporting"],
                    settings: {
                        autoAcceptMembers: false,
                        maxMembers: 50,
                    },
                },
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "Configured Team",
                    },
                ],
            },
        },
        longBio: {
            create: {
                id: validIds.id1,
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "Detailed Team",
                        bio: "This is a very detailed team biography that explains our mission, vision, values, and goals. We are committed to excellence in everything we do and strive to create meaningful impact in our field. Our team members come from diverse backgrounds and bring unique perspectives to our collaborative efforts. We believe in innovation, creativity, and continuous learning as the foundations of our success.",
                    },
                ],
            },
        },
        maxLengthFields: {
            create: {
                id: validIds.id1,
                handle: "max_length_16ch", // Exactly 16 characters (max)
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "X".repeat(50), // Max length name (50 chars)
                    },
                ],
            },
        },
        updateWithTagOperations: {
            update: {
                id: validIds.id1,
                tagsConnect: [validIds.id2],
                tagsDisconnect: [validIds.id3],
                tagsCreate: [
                    {
                        id: validIds.id4,
                        tag: "updated",
                    },
                ],
            },
        },
        updateWithMemberOperations: {
            update: {
                id: validIds.id1,
                memberInvitesCreate: [
                    {
                        id: validIds.id2,
                        message: "Updated invitation",
                        teamConnect: validIds.id1,
                        userConnect: validIds.id3,
                    },
                ],
                memberInvitesDelete: [validIds.id4],
                membersDelete: [validIds.id5, validIds.id6],
            },
        },
        updateOnlyId: {
            update: {
                id: validIds.id1,
                // Only required field
            },
        },
        differentHandle: {
            create: {
                id: validIds.id1,
                handle: "unique_handle",
                translationsCreate: [
                    {
                        id: validIds.id2,
                        language: "en",
                        name: "Unique Team",
                    },
                ],
            },
        },
    },
};

// Custom factory that always generates valid IDs and required fields
const customizers = {
    create: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
        translationsCreate: base.translationsCreate || [
            {
                id: validIds.id2,
                language: "en",
                name: "Default Team",
            },
        ],
    }),
    update: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const teamTestDataFactory = new TestDataFactory(teamFixtures, customizers);
