import type { MeetingInviteCreateInput, MeetingInviteUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { meetingInviteValidation } from "../../../validation/models/meetingInvite.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
};

// Shared meeting invite test fixtures
export const meetingInviteFixtures: ModelTestFixtures<MeetingInviteCreateInput, MeetingInviteUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            meetingConnect: validIds.id2,
            userConnect: validIds.id3,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            message: "You're invited to our weekly team meeting. Please join us to discuss the upcoming project milestones.",
            meetingConnect: validIds.id2,
            userConnect: validIds.id3,
        },
        update: {
            id: validIds.id1,
            message: "Updated invitation message with new meeting details and agenda items.",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                id: validIds.id1,
                // Missing required meetingConnect and userConnect
                message: "Join our meeting!",
            },
            update: {
                // Missing required id
                message: "Updated message",
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                message: true, // Should be string
                meetingConnect: validIds.id2,
                userConnect: validIds.id3,
            },
            update: {
                id: validIds.id1,
                message: 123, // Should be string
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                meetingConnect: validIds.id2,
                userConnect: validIds.id3,
            },
            update: {
                id: "invalid-id",
            },
        },
        missingMeetingConnect: {
            create: {
                id: validIds.id1,
                // Missing required meetingConnect
                userConnect: validIds.id3,
                message: "Please join the meeting",
            },
        },
        missingUserConnect: {
            create: {
                id: validIds.id1,
                meetingConnect: validIds.id2,
                // Missing required userConnect
                message: "You're invited!",
            },
        },
        tooLongMessage: {
            create: {
                id: validIds.id1,
                message: "A".repeat(4097), // Too long (max 4096 chars)
                meetingConnect: validIds.id2,
                userConnect: validIds.id3,
            },
            update: {
                id: validIds.id1,
                message: "B".repeat(4097), // Too long (max 4096 chars)
            },
        },
    },
    edgeCases: {
        emptyMessage: {
            create: {
                id: validIds.id1,
                message: "",
                meetingConnect: validIds.id2,
                userConnect: validIds.id3,
            },
            update: {
                id: validIds.id1,
                message: "",
            },
        },
        whitespaceMessage: {
            create: {
                id: validIds.id1,
                message: "   ", // Should be trimmed
                meetingConnect: validIds.id2,
                userConnect: validIds.id3,
            },
        },
        maxLengthMessage: {
            create: {
                id: validIds.id1,
                message: "A".repeat(4096), // Max valid message (4096 chars)
                meetingConnect: validIds.id2,
                userConnect: validIds.id3,
            },
            update: {
                id: validIds.id1,
                message: "B".repeat(4096), // Max valid message (4096 chars)
            },
        },
        multilineMessage: {
            create: {
                id: validIds.id1,
                message: `Dear Team Member,

You are cordially invited to our upcoming meeting.

Agenda:
- Project updates
- Budget review
- Q&A session

Best regards,
The Team`,
                meetingConnect: validIds.id2,
                userConnect: validIds.id3,
            },
        },
        specialCharactersMessage: {
            create: {
                id: validIds.id1,
                message: "Join us! 🎉 Meeting @ 3:00 PM <important> Don't forget & bring your notes 📝",
                meetingConnect: validIds.id2,
                userConnect: validIds.id3,
            },
        },
        differentUserAndMeetingIds: {
            create: {
                id: validIds.id1,
                meetingConnect: validIds.id4,
                userConnect: validIds.id5,
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: Partial<MeetingInviteCreateInput>): MeetingInviteCreateInput => ({
        ...base,
        id: base.id || validIds.id1,
        meetingConnect: base.meetingConnect || validIds.id2,
        userConnect: base.userConnect || validIds.id3,
    }),
    update: (base: Partial<MeetingInviteUpdateInput>): MeetingInviteUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const meetingInviteTestDataFactory = new TypedTestDataFactory(meetingInviteFixtures, meetingInviteValidation, customizers);

// Export typed fixtures for direct use in tests
export const typedMeetingInviteFixtures = createTypedFixtures(meetingInviteFixtures, meetingInviteValidation);
