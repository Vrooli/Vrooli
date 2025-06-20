// AI_CHECK: TEST_COVERAGE=2 | LAST: 2025-06-18
import { describe, expect, it } from "vitest";
import { CommentFor, RunStatus } from "../../../api/types.js";
import {
    shapeBookmark,
    shapeBookmarkList,
    shapeBot,
    shapeChat,
    shapeChatInvite,
    shapeChatMessage,
    shapeChatParticipant,
    shapeChatTranslation,
    shapeComment,
    shapeCommentTranslation,
    shapeIssue,
    shapeIssueTranslation,
    shapeMeeting,
    shapeMeetingInvite,
    shapeMeetingTranslation,
    shapeMember,
    shapeMemberInvite,
    shapeProfile,
    shapeProfileTranslation,
    shapePullRequest,
    shapePullRequestTranslation,
    shapeReminder,
    shapeReminderItem,
    shapeReminderList,
    shapeReport,
    shapeReportResponse,
    shapeResource,
    shapeResourceVersion,
    shapeResourceVersionRelation,
    shapeResourceVersionTranslation,
    shapeRun,
    shapeRunIO,
    shapeRunStep,
    shapeSchedule,
    shapeScheduleException,
    shapeScheduleRecurrence,
    shapeTag,
    shapeTagTranslation,
    shapeTeam,
    shapeTeamTranslation,
    shapeUserTranslation,
} from "../models.js";

describe("shape models", () => {
    describe("shapeBookmark", () => {
        it("should create bookmark shape", () => {
            const input = {
                id: "123",
                to: { __typename: "ResourceVersion" as const, id: "456" },
            };
            const result = shapeBookmark.create(input);
            expect(result).toEqual({
                forConnect: "456",
                bookmarkFor: "ResourceVersion",
                id: "123",
            });
        });

        it("should update bookmark shape", () => {
            const original = { id: "123", to: { __typename: "ResourceVersion" as const, id: "456" } };
            const update = { id: "123" };
            const result = shapeBookmark.update(original, update);
            expect(result).toEqual({
                id: "123",
            });
        });
    });

    describe("shapeBookmarkList", () => {
        it("should create bookmark list shape", () => {
            const input = {
                id: "123",
                label: "My Bookmarks",
                bookmarks: [{ id: "456", to: { __typename: "ResourceVersion" as const, id: "789" } }],
            };
            const result = shapeBookmarkList.create(input);
            expect(result).toEqual({
                id: "123",
                label: "My Bookmarks",
                bookmarksCreate: [{
                    forConnect: "789",
                    bookmarkFor: "ResourceVersion",
                    id: "456",
                }],
            });
        });

        it("should update bookmark list shape", () => {
            const original = { id: "123", label: "My Bookmarks" };
            const update = { id: "123", label: "Updated Bookmarks" };
            const result = shapeBookmarkList.update(original, update);
            expect(result).toEqual({
                id: "123",
                label: "Updated Bookmarks",
            });
        });
    });

    describe("shapeBot", () => {
        it("should create bot shape", () => {
            const input = {
                id: "123",
                handle: "testbot",
                name: "Test Bot",
                isBot: true as const,
                isPrivate: false,
                isBotDepictingPerson: false,
            };
            const result = shapeBot.create(input);
            expect(result).toEqual({
                id: "123",
                handle: "testbot",
                name: "Test Bot",
                isBot: true,
                isPrivate: false,
                isBotDepictingPerson: false,
                botSettings: { __version: "1.0" },
            });
        });

        it("should update bot shape", () => {
            const original = {
                id: "123",
                handle: "testbot",
                name: "Test Bot",
                isBot: true as const,
                isPrivate: false,
                isBotDepictingPerson: false,
                botSettings: { __version: "1.0" },
            };
            const update = { id: "123", name: "Updated Bot" };
            const result = shapeBot.update(original, update);
            expect(result).toEqual({
                id: "123",
                name: "Updated Bot",
                botSettings: { __version: "1.0" },
            });
        });
    });

    describe("shapeChat", () => {
        it("should create chat shape", () => {
            const input = {
                id: "123",
                openToAnyoneWithInvite: true,
            };
            const result = shapeChat.create(input);
            expect(result).toEqual({
                id: "123",
                openToAnyoneWithInvite: true,
            });
        });

        it("should update chat shape", () => {
            const original = { id: "123", openToAnyoneWithInvite: true };
            const update = { id: "123", openToAnyoneWithInvite: false };
            const result = shapeChat.update(original, update);
            expect(result).toEqual({
                id: "123",
                openToAnyoneWithInvite: false,
            });
        });
    });

    describe("shapeChatInvite", () => {
        it("should create chat invite shape", () => {
            const input = {
                id: "123",
                message: "Join our chat!",
                user: { __typename: "User" as const, id: "456" },
            };
            const result = shapeChatInvite.create(input);
            expect(result).toEqual({
                id: "123",
                message: "Join our chat!",
                userConnect: "456",
            });
        });

        it("should update chat invite shape", () => {
            const original = { id: "123", message: "Join our chat!" };
            const update = { id: "123", message: "Updated invitation!" };
            const result = shapeChatInvite.update(original, update);
            expect(result).toEqual({
                id: "123",
                message: "Updated invitation!",
            });
        });
    });

    describe("shapeChatMessage", () => {
        it("should create chat message shape", () => {
            const input = {
                id: "123",
                text: "Hello world",
                language: "en",
                versionIndex: 0,
            };
            const result = shapeChatMessage.create(input);
            expect(result).toEqual({
                id: "123",
                text: "Hello world",
                language: "en",
                versionIndex: 0,
            });
        });

        it("should update chat message shape", () => {
            const original = { id: "123", text: "Hello world", language: "en", versionIndex: 0 };
            const update = { id: "123", text: "Updated message" };
            const result = shapeChatMessage.update(original, update);
            expect(result).toEqual({
                id: "123",
                text: "Updated message",
            });
        });
    });

    describe("shapeChatParticipant", () => {
        it("should update chat participant shape", () => {
            const original = {
                id: "123",
                user: {
                    __typename: "User" as const,
                    updatedAt: "2023-01-01",
                    handle: "user123",
                    id: "456",
                    isBot: false,
                    name: "User",
                    profileImage: "image.jpg",
                    publicId: "pub123",
                },
            };
            const update = { id: "123" };
            const result = shapeChatParticipant.update(original, update);
            expect(result).toEqual({ id: "123" });
        });
    });

    describe("shapeComment", () => {
        it("should create comment shape", () => {
            const input = {
                id: "123",
                commentedOn: { __typename: "ResourceVersion" as const, id: "456" },
                translations: [{
                    id: "trans123",
                    language: "en",
                    text: "Great resource!",
                }],
            };
            const result = shapeComment.create(input);
            expect(result).toEqual({
                id: "123",
                createdFor: CommentFor.ResourceVersion,
                forConnect: "456",
                translationsCreate: [{
                    id: "trans123",
                    language: "en",
                    text: "Great resource!",
                }],
            });
        });

        it("should update comment shape", () => {
            const original = { id: "123" };
            const update = { id: "123" };
            const result = shapeComment.update(original, update);
            expect(result).toEqual({
                id: "123",
            });
        });
    });

    describe("shapeIssue", () => {
        it("should create issue shape", () => {
            const input = {
                id: "123",
                issueFor: "ResourceVersion" as const,
                for: { id: "456" },
                translations: [{
                    id: "trans123",
                    language: "en",
                    name: "Bug Report",
                    description: "Found a bug",
                }],
            };
            const result = shapeIssue.create(input);
            expect(result).toEqual({
                id: "123",
                issueFor: "ResourceVersion",
                forConnect: "456",
                translationsCreate: [{
                    id: "trans123",
                    language: "en",
                    name: "Bug Report",
                    description: "Found a bug",
                }],
            });
        });
    });

    describe("shapeMeeting", () => {
        it("should create meeting shape", () => {
            const input = {
                id: "123",
                openToAnyoneWithInvite: true,
                showOnTeamProfile: true,
                team: { id: "456" },
            };
            const result = shapeMeeting.create(input);
            expect(result).toEqual({
                id: "123",
                openToAnyoneWithInvite: true,
                showOnTeamProfile: true,
                teamConnect: "456",
            });
        });
    });

    describe("shapeMeetingInvite", () => {
        it("should create meeting invite shape", () => {
            const input = {
                id: "123",
                message: "Join our meeting!",
                meeting: { id: "456" },
                user: { id: "789" },
            };
            const result = shapeMeetingInvite.create(input);
            expect(result).toEqual({
                id: "123",
                message: "Join our meeting!",
                meetingConnect: "456",
                userConnect: "789",
            });
        });
    });

    describe("shapeMember", () => {
        it("should update member shape", () => {
            const original = {
                id: "123",
                isAdmin: false,
                permissions: ["read"],
                user: {
                    updatedAt: "2023-01-01",
                    handle: "user123",
                    id: "456",
                    isBot: false,
                    name: "User",
                    profileImage: "image.jpg",
                },
            };
            const update = { id: "123", isAdmin: true };
            const result = shapeMember.update(original, update);
            expect(result).toEqual({
                id: "123",
                isAdmin: true,
            });
        });
    });

    describe("shapeMemberInvite", () => {
        it("should create member invite shape", () => {
            const input = {
                id: "123",
                message: "Join our team!",
                willBeAdmin: false,
                willHavePermissions: ["read"],
                team: { id: "456" },
                user: {
                    __typename: "User" as const,
                    updatedAt: "2023-01-01",
                    handle: "user123",
                    id: "789",
                    isBot: false,
                    name: "User",
                    profileImage: "image.jpg",
                },
            };
            const result = shapeMemberInvite.create(input);
            expect(result).toEqual({
                id: "123",
                message: "Join our team!",
                willBeAdmin: false,
                willHavePermissions: ["read"],
                teamConnect: "456",
                userConnect: "789",
            });
        });
    });

    describe("shapeProfile", () => {
        it("should update profile shape", () => {
            const original = {
                __typename: "User" as const,
                id: "123",
                handle: "user123",
                name: "User",
                isPrivate: false,
            };
            const update = {
                id: "123",
                name: "Updated User",
                theme: "dark",
            };
            const result = shapeProfile.update(original, update);
            expect(result).toEqual({
                name: "Updated User",
                theme: "dark",
            });
        });
    });

    describe("shapePullRequest", () => {
        it("should create pull request shape", () => {
            const input = { id: "123" };
            const result = shapePullRequest.create(input);
            expect(result).toEqual({});
        });

        it("should update pull request shape", () => {
            const original = { id: "123" };
            const update = { id: "123" };
            const result = shapePullRequest.update(original, update);
            expect(result).toEqual({});
        });
    });

    describe("shapeReminder", () => {
        it("should create reminder shape", () => {
            const input = {
                id: "123",
                name: "Task",
                description: "Do something",
                dueDate: new Date("2024-01-01"),
                index: 0,
                reminderList: { id: "456" },
            };
            const result = shapeReminder.create(input);
            expect(result).toEqual({
                id: "123",
                name: "Task",
                description: "Do something",
                dueDate: expect.any(String),
                index: 0,
                reminderListConnect: "456",
            });
        });
    });

    describe("shapeReminderItem", () => {
        it("should create reminder item shape", () => {
            const input = {
                id: "123",
                name: "Subtask",
                description: "Do something specific",
                index: 0,
                reminder: { id: "456" },
            };
            const result = shapeReminderItem.create(input);
            expect(result).toEqual({
                id: "123",
                name: "Subtask",
                description: "Do something specific",
                dueDate: null,
                index: 0,
                reminderConnect: "456",
            });
        });

        it("should update reminder item shape", () => {
            const original = { id: "123", name: "Subtask", isComplete: false };
            const update = { id: "123", name: "Updated Subtask", isComplete: true };
            const result = shapeReminderItem.update(original, update);
            expect(result).toEqual({
                id: "123",
                name: "Updated Subtask",
                isComplete: true,
                dueDate: null,
            });
        });
    });

    describe("shapeReminderList", () => {
        it("should create reminder list shape", () => {
            const input = {
                id: "123",
            };
            const result = shapeReminderList.create(input);
            expect(result).toEqual({
                id: "123",
            });
        });
    });

    describe("shapeReport", () => {
        it("should create report shape", () => {
            const input = {
                id: "123",
                details: "Inappropriate content",
                language: "en",
                reason: "spam",
                createdFor: { __typename: "ResourceVersion" as const, id: "456" },
            };
            const result = shapeReport.create(input);
            expect(result).toEqual({
                id: "123",
                details: "Inappropriate content",
                language: "en",
                reason: "spam",
                createdForConnect: "456",
                createdForType: "ResourceVersion",
            });
        });
    });

    describe("shapeReportResponse", () => {
        it("should create report response shape", () => {
            const input = { id: "123" };
            const result = shapeReportResponse.create(input);
            expect(result).toEqual({});
        });
    });

    describe("shapeResource", () => {
        it("should create resource shape", () => {
            const input = {
                id: "123",
                isInternal: false,
                isPrivate: false,
                permissions: "read",
                resourceType: "pdf",
                owner: { __typename: "User" as const, id: "456" },
            };
            const result = shapeResource.create(input);
            expect(result).toEqual({
                id: "123",
                isInternal: false,
                isPrivate: false,
                permissions: "read",
                resourceType: "pdf",
                ownedByUserConnect: "456",
            });
        });
    });

    describe("shapeResourceVersion", () => {
        it("should create resource version shape", () => {
            const input = {
                id: "123",
                codeLanguage: "javascript",
                isAutomatable: true,
                isComplete: true,
                isPrivate: false,
                resourceSubType: "code",
                versionLabel: "v1.0.0",
            };
            const result = shapeResourceVersion.create(input);
            expect(result).toEqual({
                id: "123",
                codeLanguage: "javascript",
                isAutomatable: true,
                isComplete: true,
                isPrivate: false,
                resourceSubType: "code",
                versionLabel: "v1.0.0",
            });
        });
    });

    describe("shapeResourceVersionRelation", () => {
        it("should create resource version relation shape", () => {
            const input = {
                id: "123",
                labels: ["dependency"],
                toVersion: { id: "456" },
            };
            const result = shapeResourceVersionRelation.create(input);
            expect(result).toEqual({
                id: "123",
                labels: ["dependency"],
                toVersionConnect: "456",
            });
        });
    });

    describe("shapeRun", () => {
        it("should create run shape", () => {
            const input = {
                id: "123",
                isPrivate: false,
                name: "Test Run",
                status: RunStatus.InProgress,
                resourceVersion: { id: "456" },
            };
            const result = shapeRun.create(input);
            expect(result).toEqual({
                id: "123",
                isPrivate: false,
                name: "Test Run",
                status: RunStatus.InProgress,
                resourceVersionConnect: "456",
            });
        });
    });

    describe("shapeRunIO", () => {
        it("should create run IO shape", () => {
            const input = {
                id: "123",
                data: { key: "value" },
                nodeInputName: "input1",
                nodeName: "node1",
                run: { id: "456" },
            };
            const result = shapeRunIO.create(input);
            expect(result).toEqual({
                id: "123",
                data: { key: "value" },
                nodeInputName: "input1",
                nodeName: "node1",
                runConnect: "456",
            });
        });
    });

    describe("shapeRunStep", () => {
        it("should create run step shape", () => {
            const input = {
                id: "123",
                name: "Step 1",
                nodeId: "node1",
                order: 1,
                status: RunStatus.InProgress,
                run: { id: "456" },
            };
            const result = shapeRunStep.create(input);
            expect(result).toEqual({
                id: "123",
                name: "Step 1",
                nodeId: "node1",
                order: 1,
                status: RunStatus.InProgress,
                runConnect: "456",
            });
        });
    });

    describe("shapeSchedule", () => {
        it("should create schedule shape", () => {
            const input = {
                id: "123",
                startTime: new Date("2024-01-01"),
                timezone: "UTC",
            };
            const result = shapeSchedule.create(input);
            expect(result).toEqual({
                id: "123",
                startTime: expect.any(String),
                endTime: null,
                timezone: "UTC",
            });
        });
    });

    describe("shapeScheduleException", () => {
        it("should create schedule exception shape", () => {
            const input = {
                id: "123",
                originalStartTime: new Date("2024-01-01"),
                newStartTime: new Date("2024-01-02"),
                schedule: { id: "456" },
            };
            const result = shapeScheduleException.create(input);
            expect(result).toEqual({
                id: "123",
                originalStartTime: expect.any(String),
                newStartTime: expect.any(String),
                newEndTime: null,
                scheduleConnect: "456",
            });
        });
    });

    describe("shapeScheduleRecurrence", () => {
        it("should create schedule recurrence shape", () => {
            const input = {
                id: "123",
                recurrenceType: "daily",
                interval: 1,
                schedule: { id: "456" },
            };
            const result = shapeScheduleRecurrence.create(input);
            expect(result).toEqual({
                id: "123",
                recurrenceType: "daily",
                interval: 1,
                endDate: null,
                scheduleConnect: "456",
            });
        });
    });

    describe("shapeTag", () => {
        it("should create tag shape", () => {
            const input = {
                id: "123",
                tag: "javascript",
            };
            const result = shapeTag.create(input);
            expect(result).toEqual({
                id: "123",
                tag: "javascript",
            });
        });

        it("should use tag as id field", () => {
            expect(shapeTag.idField).toBe("tag");
        });
    });

    describe("shapeTeam", () => {
        it("should create team shape", () => {
            const input = {
                id: "123",
                handle: "team123",
                isOpenToNewMembers: true,
                isPrivate: false,
            };
            const result = shapeTeam.create(input);
            expect(result).toEqual({
                id: "123",
                handle: "team123",
                isOpenToNewMembers: true,
                isPrivate: false,
            });
        });
    });

    describe("Translation shapers", () => {
        it("should create user translation shape", () => {
            const input = {
                id: "123",
                language: "en",
                bio: "User bio",
            };
            const result = shapeUserTranslation.create(input);
            expect(result).toEqual({
                id: "123",
                language: "en",
                bio: "User bio",
            });
        });

        it("should create chat translation shape", () => {
            const input = {
                id: "123",
                language: "en",
                name: "Chat Name",
                description: "Chat Description",
            };
            const result = shapeChatTranslation.create(input);
            expect(result).toEqual({
                id: "123",
                language: "en",
                name: "Chat Name",
                description: "Chat Description",
            });
        });

        it("should create comment translation shape", () => {
            const input = {
                id: "123",
                language: "en",
                text: "Comment text",
            };
            const result = shapeCommentTranslation.create(input);
            expect(result).toEqual({
                id: "123",
                language: "en",
                text: "Comment text",
            });
        });

        it("should create issue translation shape", () => {
            const input = {
                id: "123",
                language: "en",
                name: "Issue Name",
                description: "Issue Description",
            };
            const result = shapeIssueTranslation.create(input);
            expect(result).toEqual({
                id: "123",
                language: "en",
                name: "Issue Name",
                description: "Issue Description",
            });
        });

        it("should create meeting translation shape", () => {
            const input = {
                id: "123",
                language: "en",
                name: "Meeting Name",
                description: "Meeting Description",
                link: "https://meet.example.com",
            };
            const result = shapeMeetingTranslation.create(input);
            expect(result).toEqual({
                id: "123",
                language: "en",
                name: "Meeting Name",
                description: "Meeting Description",
                link: "https://meet.example.com",
            });
        });

        it("should create profile translation shape", () => {
            const input = {
                id: "123",
                language: "en",
                bio: "Profile bio",
            };
            const result = shapeProfileTranslation.create(input);
            expect(result).toEqual({
                id: "123",
                language: "en",
                bio: "Profile bio",
            });
        });

        it("should create pull request translation shape", () => {
            const input = {
                id: "123",
                language: "en",
                text: "PR description",
            };
            const result = shapePullRequestTranslation.create(input);
            expect(result).toEqual({
                id: "123",
                language: "en",
                text: "PR description",
            });
        });

        it("should create resource version translation shape", () => {
            const input = {
                id: "123",
                language: "en",
                name: "Resource Name",
                description: "Resource Description",
            };
            const result = shapeResourceVersionTranslation.create(input);
            expect(result).toEqual({
                id: "123",
                language: "en",
                name: "Resource Name",
                description: "Resource Description",
                instructions: "",
            });
        });

        it("should create tag translation shape", () => {
            const input = {
                id: "123",
                language: "en",
                description: "Tag description",
            };
            const result = shapeTagTranslation.create(input);
            expect(result).toEqual({
                id: "123",
                language: "en",
                description: "Tag description",
            });
        });

        it("should create team translation shape", () => {
            const input = {
                id: "123",
                language: "en",
                name: "Team Name",
                bio: "Team bio",
            };
            const result = shapeTeamTranslation.create(input);
            expect(result).toEqual({
                id: "123",
                language: "en",
                name: "Team Name",
                bio: "Team bio",
            });
        });
    });

    describe("Translation shapers without update", () => {
        describe("shapeCommentTranslation", () => {
            it("should create comment translation shape", () => {
                const input = {
                    id: "trans123",
                    language: "en",
                    text: "Great comment!",
                };
                const result = shapeCommentTranslation.create(input);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    text: "Great comment!",
                });
            });

            it("should update comment translation shape", () => {
                const original = {
                    id: "trans123",
                    language: "en",
                    text: "Original text",
                };
                const update = {
                    id: "trans123",
                    text: "Updated text",
                };
                const result = shapeCommentTranslation.update(original, update);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    text: "Updated text",
                });
            });
        });

        describe("shapeIssueTranslation", () => {
            it("should create issue translation shape", () => {
                const input = {
                    id: "trans123",
                    language: "en",
                    name: "Bug Report",
                    description: "Description of bug",
                };
                const result = shapeIssueTranslation.create(input);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    name: "Bug Report",
                    description: "Description of bug",
                });
            });

            it("should update issue translation shape", () => {
                const original = {
                    id: "trans123",
                    language: "en",
                    name: "Bug Report",
                    description: "Original description",
                };
                const update = {
                    id: "trans123",
                    description: "Updated description",
                };
                const result = shapeIssueTranslation.update(original, update);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    description: "Updated description",
                });
            });
        });

        describe("shapeMeetingTranslation", () => {
            it("should create meeting translation shape", () => {
                const input = {
                    id: "trans123",
                    language: "en",
                    name: "Team Meeting",
                    description: "Weekly sync",
                    link: "https://meet.example.com",
                };
                const result = shapeMeetingTranslation.create(input);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    name: "Team Meeting",
                    description: "Weekly sync",
                    link: "https://meet.example.com",
                });
            });

            it("should update meeting translation shape", () => {
                const original = {
                    id: "trans123",
                    language: "en",
                    name: "Team Meeting",
                    description: "Weekly sync",
                    link: "https://meet.example.com",
                };
                const update = {
                    id: "trans123",
                    link: "https://newmeet.example.com",
                };
                const result = shapeMeetingTranslation.update(original, update);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    link: "https://newmeet.example.com",
                });
            });
        });

        describe("shapeProfileTranslation", () => {
            it("should create profile translation shape", () => {
                const input = {
                    id: "trans123",
                    language: "en",
                    bio: "Software developer",
                };
                const result = shapeProfileTranslation.create(input);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    bio: "Software developer",
                });
            });

            it("should update profile translation shape", () => {
                const original = {
                    id: "trans123",
                    language: "en",
                    bio: "Software developer",
                };
                const update = {
                    id: "trans123",
                    bio: "Senior software developer",
                };
                const result = shapeProfileTranslation.update(original, update);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    bio: "Senior software developer",
                });
            });
        });

        describe("shapePullRequestTranslation", () => {
            it("should create pull request translation shape", () => {
                const input = {
                    id: "trans123",
                    language: "en",
                    text: "Fix bug in authentication",
                };
                const result = shapePullRequestTranslation.create(input);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    text: "Fix bug in authentication",
                });
            });

            it("should update pull request translation shape", () => {
                const original = {
                    id: "trans123",
                    language: "en",
                    text: "Fix bug in authentication",
                };
                const update = {
                    id: "trans123",
                    text: "Fix critical bug in authentication",
                };
                const result = shapePullRequestTranslation.update(original, update);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    text: "Fix critical bug in authentication",
                });
            });
        });

        describe("shapeResourceVersionTranslation", () => {
            it("should create resource version translation shape", () => {
                const input = {
                    id: "trans123",
                    language: "en",
                    name: "API Documentation",
                    description: "Complete API reference",
                    instructions: "Follow these steps",
                };
                const result = shapeResourceVersionTranslation.create(input);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    name: "API Documentation",
                    description: "Complete API reference",
                    instructions: "Follow these steps",
                });
            });

            it("should update resource version translation shape", () => {
                const original = {
                    id: "trans123",
                    language: "en",
                    name: "API Documentation",
                    description: "Complete API reference",
                    instructions: "Follow these steps",
                };
                const update = {
                    id: "trans123",
                    instructions: "Updated instructions",
                };
                const result = shapeResourceVersionTranslation.update(original, update);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    instructions: "Updated instructions",
                });
            });

            it("should handle missing instructions field", () => {
                const input = {
                    id: "trans123",
                    language: "en",
                    name: "API Documentation",
                    description: "Complete API reference",
                };
                const result = shapeResourceVersionTranslation.create(input);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    name: "API Documentation",
                    description: "Complete API reference",
                    instructions: "", // Should default to empty string
                });
            });
        });

        describe("shapeTagTranslation", () => {
            it("should create tag translation shape", () => {
                const input = {
                    id: "trans123",
                    language: "en",
                    description: "JavaScript programming language",
                };
                const result = shapeTagTranslation.create(input);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    description: "JavaScript programming language",
                });
            });

            it("should update tag translation shape", () => {
                const original = {
                    id: "trans123",
                    language: "en",
                    description: "JavaScript programming language",
                };
                const update = {
                    id: "trans123",
                    description: "JavaScript - a dynamic programming language",
                };
                const result = shapeTagTranslation.update(original, update);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    description: "JavaScript - a dynamic programming language",
                });
            });
        });

        describe("shapeTeamTranslation", () => {
            it("should create team translation shape", () => {
                const input = {
                    id: "trans123",
                    language: "en",
                    name: "Development Team",
                    bio: "We build awesome software",
                };
                const result = shapeTeamTranslation.create(input);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    name: "Development Team",
                    bio: "We build awesome software",
                });
            });

            it("should update team translation shape", () => {
                const original = {
                    id: "trans123",
                    language: "en",
                    name: "Development Team",
                    bio: "We build awesome software",
                };
                const update = {
                    id: "trans123",
                    bio: "We build innovative software solutions",
                };
                const result = shapeTeamTranslation.update(original, update);
                expect(result).toEqual({
                    id: "trans123",
                    language: "en",
                    bio: "We build innovative software solutions",
                });
            });
        });
    });
});
