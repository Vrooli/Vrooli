import type { MeetingCreateInput, MeetingUpdateInput, MeetingTranslationCreateInput, MeetingTranslationUpdateInput, MeetingInviteCreateInput, MeetingInviteUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { meetingValidation } from "../../../validation/models/meeting.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
};

// Shared meeting test fixtures
export const meetingFixtures: ModelTestFixtures<MeetingCreateInput, MeetingUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            teamConnect: validIds.id2,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            openToAnyoneWithInvite: true,
            showOnTeamProfile: false,
            teamConnect: validIds.id2,
            translationsCreate: [{
                id: validIds.id3,
                language: "en",
                name: "Weekly Team Meeting",
                description: "Our regular weekly synchronization meeting to discuss progress and planning",
                link: "https://example.com/meeting",
            }],
            invitesCreate: [{
                id: validIds.id4,
                message: "You're invited to our weekly team meeting",
                meetingConnect: validIds.id1,
                userConnect: validIds.id5,
            }],
            // Remove schedule from complete fixture - test schedule separately
        },
        update: {
            id: validIds.id1,
            openToAnyoneWithInvite: false,
            showOnTeamProfile: true,
            translationsCreate: [{
                id: validIds.id3,
                language: "es",
                name: "Reunión Semanal del Equipo",
                description: "Nuestra reunión semanal regular para discutir el progreso y la planificación",
                link: "https://example.com/meeting-es",
            }],
            translationsUpdate: [{
                id: validIds.id4,
                language: "en",
                name: "Updated Weekly Team Meeting",
                description: "Updated description for our weekly meeting",
            }],
            translationsDelete: [validIds.id5],
            invitesCreate: [{
                id: validIds.id2,
                message: "New team member invitation",
                meetingConnect: validIds.id1,
                userConnect: validIds.id3,
            }],
            invitesUpdate: [{
                id: validIds.id4,
                message: "Updated invitation message",
            }],
            invitesDelete: [validIds.id5],
            // Remove schedule from complete fixture - test schedule separately
        },
    },
    invalid: {
        missingRequired: {
            create: {
                id: validIds.id1,
                // Missing required teamConnect
                openToAnyoneWithInvite: true,
            },
            update: {
                // Missing required id
                openToAnyoneWithInvite: false,
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                openToAnyoneWithInvite: "maybe", // Should be boolean
                showOnTeamProfile: 1, // Should be boolean
                teamConnect: validIds.id2,
            },
            update: {
                id: validIds.id1,
                openToAnyoneWithInvite: "true", // Will be converted to boolean
                showOnTeamProfile: "false", // Will be converted to boolean
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                teamConnect: validIds.id2,
            },
            update: {
                id: "invalid-id",
            },
        },
        invalidTranslations: {
            create: {
                id: validIds.id1,
                teamConnect: validIds.id2,
                translationsCreate: [{
                    id: validIds.id3,
                    language: "en",
                    name: "AB", // Too short (min 3 chars)
                    description: "Valid description",
                }],
            },
        },
        invalidTranslationUrl: {
            create: {
                id: validIds.id1,
                teamConnect: validIds.id2,
                translationsCreate: [{
                    id: validIds.id3,
                    language: "en",
                    name: "Valid Meeting Name",
                    link: "not-a-valid-url",
                }],
            },
        },
        tooLongTranslationName: {
            create: {
                id: validIds.id1,
                teamConnect: validIds.id2,
                translationsCreate: [{
                    id: validIds.id3,
                    language: "en",
                    name: "A".repeat(51), // Too long (max 50 chars)
                    description: "Valid description",
                }],
            },
        },
        tooLongTranslationDescription: {
            create: {
                id: validIds.id1,
                teamConnect: validIds.id2,
                translationsCreate: [{
                    id: validIds.id3,
                    language: "en",
                    name: "Valid Meeting Name",
                    description: "A".repeat(2049), // Too long (max 2048 chars)
                }],
            },
        },
        invalidInvite: {
            create: {
                id: validIds.id1,
                teamConnect: validIds.id2,
                invitesCreate: [{
                    id: validIds.id3,
                    // Missing required meetingConnect and userConnect
                    message: "Valid message",
                }],
            },
        },
    },
    edgeCases: {
        withOptionalFields: {
            create: {
                id: validIds.id1,
                teamConnect: validIds.id2,
                openToAnyoneWithInvite: false,
                showOnTeamProfile: true,
            },
        },
        booleanConversions: {
            create: {
                id: validIds.id1,
                teamConnect: validIds.id2,
                openToAnyoneWithInvite: "true", // String to boolean
                showOnTeamProfile: "no", // String to boolean
            },
        },
        emptyTranslations: {
            create: {
                id: validIds.id1,
                teamConnect: validIds.id2,
                translationsCreate: [],
            },
        },
        multipleTranslations: {
            create: {
                id: validIds.id1,
                teamConnect: validIds.id2,
                translationsCreate: [
                    {
                        id: validIds.id3,
                        language: "en",
                        name: "English Meeting",
                        description: "Meeting description in English",
                        link: "https://example.com/en",
                    },
                    {
                        id: validIds.id4,
                        language: "es",
                        name: "Reunión en Español",
                        description: "Descripción de la reunión en español",
                        link: "https://example.com/es",
                    },
                ],
            },
        },
        withScheduleAndInvites: {
            create: {
                id: validIds.id1,
                teamConnect: validIds.id2,
                openToAnyoneWithInvite: true,
                // Remove schedule from edge case - test schedule separately
                invitesCreate: [{
                    id: validIds.id4,
                    message: "Join our important meeting",
                    meetingConnect: validIds.id1,
                    userConnect: validIds.id5,
                }],
            },
        },
        minimalTranslation: {
            create: {
                id: validIds.id1,
                teamConnect: validIds.id2,
                translationsCreate: [{
                    id: validIds.id3,
                    language: "en",
                    name: "Min", // Minimum valid name (3 chars)
                }],
            },
        },
        maxLengthTranslation: {
            create: {
                id: validIds.id1,
                teamConnect: validIds.id2,
                translationsCreate: [{
                    id: validIds.id3,
                    language: "en",
                    name: "A".repeat(50), // Max valid name (50 chars)
                    description: "A".repeat(2048), // Max valid description (2048 chars)
                    link: "https://example.com/meeting", // Valid URL
                }],
            },
        },
    },
};

// Custom factory for creating test data programmatically
const customizers = {
    create: (base: Partial<MeetingCreateInput>): MeetingCreateInput => ({
        ...base,
        id: base.id || validIds.id1,
        teamConnect: base.teamConnect || validIds.id2,
    }),
    update: (base: Partial<MeetingUpdateInput>): MeetingUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const meetingTestDataFactory = new TypedTestDataFactory(meetingFixtures, meetingValidation, customizers);
export const typedMeetingFixtures = createTypedFixtures(meetingFixtures, meetingValidation);

// For backward compatibility, also export non-typed factory
export const meetingTestDataFactoryUntyped = new TestDataFactory(meetingFixtures, customizers);
