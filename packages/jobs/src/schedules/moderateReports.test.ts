// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-24
import { ReportStatus, ReportSuggestedAction, generatePK, generatePublicId } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { moderateReports } from "./moderateReports.js";

const { DbProvider } = await import("@vrooli/server");

// Mock the Trigger function only
vi.mock("@vrooli/server", async () => {
    const actual = await vi.importActual("@vrooli/server");
    return {
        ...actual,
        Trigger: vi.fn(() => ({
            reportActivity: vi.fn().mockResolvedValue(undefined),
        })),
    };
});

describe("moderateReports integration tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testTeamIds: bigint[] = [];
    const testCommentIds: bigint[] = [];
    const testIssueIds: bigint[] = [];
    const testTagIds: bigint[] = [];
    const testReportIds: bigint[] = [];
    const testReportResponseIds: bigint[] = [];
    const testRoutineIds: bigint[] = [];
    const testRoutineVersionIds: bigint[] = [];
    const testApiIds: bigint[] = [];
    const testApiVersionIds: bigint[] = [];

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testTeamIds.length = 0;
        testCommentIds.length = 0;
        testIssueIds.length = 0;
        testTagIds.length = 0;
        testReportIds.length = 0;
        testReportResponseIds.length = 0;
        testRoutineIds.length = 0;
        testRoutineVersionIds.length = 0;
        testApiIds.length = 0;
        testApiVersionIds.length = 0;

        // Reset mocks
        vi.clearAllMocks();
    });

    afterEach(async () => {
        // Clean up test data
        const db = DbProvider.get();
        
        // Clean up in reverse dependency order
        if (testReportResponseIds.length > 0) {
            await db.report_response.deleteMany({ where: { id: { in: testReportResponseIds } } });
        }
        if (testReportIds.length > 0) {
            await db.report.deleteMany({ where: { id: { in: testReportIds } } });
        }
        if (testApiVersionIds.length > 0) {
            await db.resource_version.deleteMany({ where: { id: { in: testApiVersionIds } } });
        }
        if (testApiIds.length > 0) {
            await db.resource.deleteMany({ where: { id: { in: testApiIds } } });
        }
        if (testRoutineVersionIds.length > 0) {
            await db.resource_version.deleteMany({ where: { id: { in: testRoutineVersionIds } } });
        }
        if (testRoutineIds.length > 0) {
            await db.resource.deleteMany({ where: { id: { in: testRoutineIds } } });
        }
        if (testTagIds.length > 0) {
            await db.tag.deleteMany({ where: { id: { in: testTagIds } } });
        }
        if (testIssueIds.length > 0) {
            await db.issue.deleteMany({ where: { id: { in: testIssueIds } } });
        }
        if (testCommentIds.length > 0) {
            await db.comment.deleteMany({ where: { id: { in: testCommentIds } } });
        }
        if (testTeamIds.length > 0) {
            await db.team.deleteMany({ where: { id: { in: testTeamIds } } });
        }
        if (testUserIds.length > 0) {
            await db.user.deleteMany({ where: { id: { in: testUserIds } } });
        }
    });

    it("should close report and delete object when delete action has enough reputation", async () => {
        // Create users
        const objectOwner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Object Owner",
                handle: "objectowner",
            },
        });
        testUserIds.push(objectOwner.id);

        const reporter = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Reporter",
                handle: "reporter",
            },
        });
        testUserIds.push(reporter.id);

        const highRepUser = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "High Rep User",
                handle: "highrepuser",
                reputation: 600, // Above MIN_REP.Delete (500)
            },
        });
        testUserIds.push(highRepUser.id);

        // Create a comment to report
        // Create an issue first for the comment
        const issue = await DbProvider.get().issue.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: objectOwner.id },
                },
                translations: {
                    create: {
                        id: generatePK(),
                        language: "en",
                        name: "Test Issue",
                        description: "Issue for comment",
                    },
                },
            },
        });
        testIssueIds.push(issue.id);

        const comment = await DbProvider.get().comment.create({
            data: {
                id: generatePK(),
                ownedByUser: { connect: { id: objectOwner.id } },
                issue: { connect: { id: issue.id } },
                translations: {
                    create: {
                        id: generatePK(),
                        language: "en",
                        text: "Inappropriate comment",
                    },
                },
            },
        });
        testCommentIds.push(comment.id);

        // Create report
        const report = await DbProvider.get().report.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: reporter.id },
                },
                comment: {
                    connect: { id: comment.id },
                },
                reason: "Spam",
                details: "This is spam content",
                status: ReportStatus.Open,
                language: "en",
            },
        });
        testReportIds.push(report.id);

        // Create report response suggesting delete
        const response = await DbProvider.get().report_response.create({
            data: {
                id: generatePK(),
                report: {
                    connect: { id: report.id },
                },
                createdBy: {
                    connect: { id: highRepUser.id },
                },
                actionSuggested: ReportSuggestedAction.Delete,
                details: "This should be deleted",
            },
        });
        testReportResponseIds.push(response.id);

        // Run moderation
        await moderateReports();

        // Check that comment was deleted
        const deletedComment = await DbProvider.get().comment.findUnique({
            where: { id: comment.id },
        });
        expect(deletedComment).toBeNull();

        // Note: The report is cascade deleted when the comment is deleted,
        // so we can't check the report status after deletion.
        // The fact that the comment was deleted proves the moderation worked.

        // Check that trigger was called
        const { Trigger } = await import("@vrooli/server");
        expect(Trigger).toHaveBeenCalled();
    });

    it("should hide object when HideUntilFixed action has enough reputation", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner",
                handle: "owner",
            },
        });
        testUserIds.push(owner.id);

        const reporter = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Reporter",
                handle: "reporter2",
            },
        });
        testUserIds.push(reporter.id);

        const moderator1 = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Moderator 1",
                handle: "mod1",
                reputation: 60,
            },
        });
        testUserIds.push(moderator1.id);

        const moderator2 = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Moderator 2",
                handle: "mod2",
                reputation: 50,
            },
        });
        testUserIds.push(moderator2.id);

        // Create a resource and version to report
        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: owner.id },
                },
                ownedByUser: {
                    connect: { id: owner.id },
                },
                isPrivate: false,
                isInternal: false,
                resourceType: "Routine",
            },
        });
        testRoutineIds.push(resource.id);

        const resourceVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                rootId: resource.id,
                versionLabel: "1.0.0",
                isPrivate: false,
            },
        });
        testRoutineVersionIds.push(resourceVersion.id);

        // Create report
        const report = await DbProvider.get().report.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: reporter.id },
                },
                resourceVersion: {

                    connect: { id: resourceVersion.id },

                },
                reason: "Bug",
                details: "Has a critical bug",
                status: ReportStatus.Open,
                language: "en",
            },
        });
        testReportIds.push(report.id);

        // Create report responses suggesting hide (total rep = 110, meets MIN_REP.HideUntilFixed = 100)
        const response1 = await DbProvider.get().report_response.create({
            data: {
                id: generatePK(),
                report: {
                    connect: { id: report.id },
                },
                createdBy: {
                    connect: { id: moderator1.id },
                },
                actionSuggested: ReportSuggestedAction.HideUntilFixed,
            },
        });
        testReportResponseIds.push(response1.id);

        const response2 = await DbProvider.get().report_response.create({
            data: {
                id: generatePK(),
                report: {
                    connect: { id: report.id },
                },
                createdBy: {
                    connect: { id: moderator2.id },
                },
                actionSuggested: ReportSuggestedAction.HideUntilFixed,
            },
        });
        testReportResponseIds.push(response2.id);

        await moderateReports();

        // Check that report was closed
        const updatedReport = await DbProvider.get().report.findUnique({
            where: { id: report.id },
        });
        expect(updatedReport?.status).toBe(ReportStatus.ClosedHidden);

        // Check that resource version was hidden
        const updatedVersion = await DbProvider.get().resource_version.findUnique({
            where: { id: resourceVersion.id },
        });
        expect(updatedVersion?.isPrivate).toBe(true);
    });

    it("should mark report as false when FalseReport action has enough reputation", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Innocent User",
                handle: "innocent",
                reputation: 1000,
            },
        });
        testUserIds.push(owner.id);

        const badReporter = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Bad Reporter",
                handle: "badreporter",
            },
        });
        testUserIds.push(badReporter.id);

        const reviewer = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Reviewer",
                handle: "reviewer",
                reputation: 150,
            },
        });
        testUserIds.push(reviewer.id);

        // Create an issue
        const issue = await DbProvider.get().issue.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: owner.id },
                },
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Valid Issue",
                        description: "This is a legitimate issue",
                    }],
                },
            },
        });
        testIssueIds.push(issue.id);

        // Create false report
        const report = await DbProvider.get().report.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: badReporter.id },
                },
                issue: {
                    connect: { id: issue.id },
                },
                reason: "Spam",
                details: "False accusation",
                status: ReportStatus.Open,
                language: "en",
            },
        });
        testReportIds.push(report.id);

        // Create response marking as false report
        const response = await DbProvider.get().report_response.create({
            data: {
                id: generatePK(),
                report: {
                    connect: { id: report.id },
                },
                createdBy: {
                    connect: { id: reviewer.id },
                },
                actionSuggested: ReportSuggestedAction.FalseReport,
                details: "This is clearly a false report",
            },
        });
        testReportResponseIds.push(response.id);

        await moderateReports();

        // Check that report was closed as false
        const updatedReport = await DbProvider.get().report.findUnique({
            where: { id: report.id },
        });
        expect(updatedReport?.status).toBe(ReportStatus.ClosedFalseReport);

        // Check that issue was NOT affected
        const unchangedIssue = await DbProvider.get().issue.findUnique({
            where: { id: issue.id },
        });
        expect(unchangedIssue).not.toBeNull();
        expect(unchangedIssue?.publicId).toBe(issue.publicId);
    });

    it("should choose least severe action when there's a tie in reputation", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Tag Owner",
                handle: "tagowner",
            },
        });
        testUserIds.push(owner.id);

        const reporter = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Reporter",
                handle: "reporter3",
            },
        });
        testUserIds.push(reporter.id);

        // Create users with reputation that will cause a tie
        const deleteVoter = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Delete Voter",
                handle: "deletevoter",
                reputation: 500, // Exactly MIN_REP.Delete
            },
        });
        testUserIds.push(deleteVoter.id);

        const nonIssueVoter = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "NonIssue Voter",
                handle: "nonissuevoter",
                reputation: 500, // Same total as delete
            },
        });
        testUserIds.push(nonIssueVoter.id);

        // Create a tag
        const tag = await DbProvider.get().tag.create({
            data: {
                id: generatePK(),
                createdBy: {
                    connect: { id: owner.id },
                },
                tag: "test-tag",
            },
        });
        testTagIds.push(tag.id);

        // Create report
        const report = await DbProvider.get().report.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: reporter.id },
                },
                tag: {
                    connect: { id: tag.id },
                },
                reason: "Inappropriate",
                status: ReportStatus.Open,
                language: "en",
            },
        });
        testReportIds.push(report.id);

        // Create responses with tied reputation
        const deleteResponse = await DbProvider.get().report_response.create({
            data: {
                id: generatePK(),
                report: {
                    connect: { id: report.id },
                },
                createdBy: {
                    connect: { id: deleteVoter.id },
                },
                actionSuggested: ReportSuggestedAction.Delete,
            },
        });
        testReportResponseIds.push(deleteResponse.id);

        const nonIssueResponse = await DbProvider.get().report_response.create({
            data: {
                id: generatePK(),
                report: {
                    connect: { id: report.id },
                },
                createdBy: {
                    connect: { id: nonIssueVoter.id },
                },
                actionSuggested: ReportSuggestedAction.NonIssue,
            },
        });
        testReportResponseIds.push(nonIssueResponse.id);

        await moderateReports();

        // Should choose NonIssue (least severe)
        const updatedReport = await DbProvider.get().report.findUnique({
            where: { id: report.id },
        });
        expect(updatedReport?.status).toBe(ReportStatus.ClosedNonIssue);

        // Tag should still exist
        const existingTag = await DbProvider.get().tag.findUnique({
            where: { id: tag.id },
        });
        expect(existingTag).not.toBeNull();
    });

    it("should not close report when no action meets minimum reputation", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner",
                handle: "owner2",
            },
        });
        testUserIds.push(owner.id);

        const reporter = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Reporter",
                handle: "reporter4",
            },
        });
        testUserIds.push(reporter.id);

        const lowRepUser = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Low Rep User",
                handle: "lowrep",
                reputation: 50, // Below all MIN_REP thresholds
            },
        });
        testUserIds.push(lowRepUser.id);

        // Create team to report
        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: owner.id },
                },
                handle: "testteam",
            },
        });
        testTeamIds.push(team.id);

        // Create recent report (not timed out)
        const report = await DbProvider.get().report.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: reporter.id },
                },
                team: {
                    connect: { id: team.id },
                },
                reason: "Spam",
                status: ReportStatus.Open,
                language: "en",
                createdAt: new Date(), // Recent
            },
        });
        testReportIds.push(report.id);

        // Create response with insufficient reputation
        const response = await DbProvider.get().report_response.create({
            data: {
                id: generatePK(),
                report: {
                    connect: { id: report.id },
                },
                createdBy: {
                    connect: { id: lowRepUser.id },
                },
                actionSuggested: ReportSuggestedAction.Delete,
            },
        });
        testReportResponseIds.push(response.id);

        await moderateReports();

        // Report should still be open
        const unchangedReport = await DbProvider.get().report.findUnique({
            where: { id: report.id },
        });
        expect(unchangedReport?.status).toBe(ReportStatus.Open);
    });

    it("should close report after timeout even with low reputation", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner",
                handle: "owner3",
            },
        });
        testUserIds.push(owner.id);

        const reporter = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Reporter",
                handle: "reporter5",
            },
        });
        testUserIds.push(reporter.id);

        const voter = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Voter",
                handle: "voter",
                reputation: 10, // Very low
            },
        });
        testUserIds.push(voter.id);

        // Create comment
        const comment = await DbProvider.get().comment.create({
            data: {
                id: generatePK(),
                ownedByUser: { connect: { id: owner.id } },
                translations: {
                    create: {
                        id: generatePK(),
                        language: "en",
                        text: "Old comment",
                    },
                },
            },
        });
        testCommentIds.push(comment.id);

        // Create old report (over 1 week)
        const oldDate = new Date(Date.now() - 8 * 24 * 60 * 60 * 1000); // 8 days ago
        const report = await DbProvider.get().report.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: reporter.id },
                },
                comment: {
                    connect: { id: comment.id },
                },
                reason: "Inappropriate",
                status: ReportStatus.Open,
                language: "en",
                createdAt: oldDate,
            },
        });
        testReportIds.push(report.id);

        // Create response with low reputation
        const response = await DbProvider.get().report_response.create({
            data: {
                id: generatePK(),
                report: {
                    connect: { id: report.id },
                },
                createdBy: {
                    connect: { id: voter.id },
                },
                actionSuggested: ReportSuggestedAction.HideUntilFixed,
            },
        });
        testReportResponseIds.push(response.id);

        await moderateReports();

        // Report should be closed after timeout
        const updatedReport = await DbProvider.get().report.findUnique({
            where: { id: report.id },
        });
        expect(updatedReport?.status).toBe(ReportStatus.ClosedHidden);
    });

    it("should handle reports on versioned objects correctly", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "API Owner",
                handle: "apiowner",
            },
        });
        testUserIds.push(owner.id);

        const reporter = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Reporter",
                handle: "reporter6",
            },
        });
        testUserIds.push(reporter.id);

        const moderator = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Moderator",
                handle: "moderator",
                reputation: 600,
            },
        });
        testUserIds.push(moderator.id);

        // Create resource and version (API type)
        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: owner.id },
                },
                ownedByUser: {
                    connect: { id: owner.id },
                },
                isPrivate: false,
                isInternal: false,
                resourceType: "Api",
            },
        });
        testApiIds.push(resource.id);

        const resourceVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                rootId: resource.id,
                versionLabel: "1.0.0",
                isDeleted: false,
            },
        });
        testApiVersionIds.push(resourceVersion.id);

        // Create report on API version
        const report = await DbProvider.get().report.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: reporter.id },
                },
                resourceVersion: {

                    connect: { id: resourceVersion.id },

                },
                reason: "Malicious",
                status: ReportStatus.Open,
                language: "en",
            },
        });
        testReportIds.push(report.id);

        // Create response to delete
        const response = await DbProvider.get().report_response.create({
            data: {
                id: generatePK(),
                report: {
                    connect: { id: report.id },
                },
                createdBy: {
                    connect: { id: moderator.id },
                },
                actionSuggested: ReportSuggestedAction.Delete,
            },
        });
        testReportResponseIds.push(response.id);

        await moderateReports();

        // Check that report was closed
        const updatedReport = await DbProvider.get().report.findUnique({
            where: { id: report.id },
        });
        expect(updatedReport?.status).toBe(ReportStatus.ClosedDeleted);

        // Check that resource version was soft-deleted
        const updatedVersion = await DbProvider.get().resource_version.findUnique({
            where: { id: resourceVersion.id },
        });
        expect(updatedVersion?.isDeleted).toBe(true);

        // Check that root resource was NOT deleted
        const rootResource = await DbProvider.get().resource.findUnique({
            where: { id: resource.id },
        });
        expect(rootResource).not.toBeNull();
    });

    it("should handle multiple report responses with different actions", async () => {
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner",
                handle: "owner4",
            },
        });
        testUserIds.push(owner.id);

        const reporter = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Reporter",
                handle: "reporter7",
            },
        });
        testUserIds.push(reporter.id);

        // Create voters with different reputations
        const voters = await Promise.all([
            DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Delete Voter 1",
                    handle: "deletevoter1",
                    reputation: 200,
                },
            }),
            DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Delete Voter 2",
                    handle: "deletevoter2",
                    reputation: 150,
                },
            }),
            DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Hide Voter",
                    handle: "hidevoter",
                    reputation: 300,
                },
            }),
            DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "NonIssue Voter",
                    handle: "nonissuevoter2",
                    reputation: 50,
                },
            }),
        ]);
        voters.forEach(v => testUserIds.push(v.id));

        // Create issue
        const issue = await DbProvider.get().issue.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: owner.id },
                },
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Test Issue",
                        description: "Issue for testing",
                    }],
                },
            },
        });
        testIssueIds.push(issue.id);

        // Create report
        const report = await DbProvider.get().report.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: reporter.id },
                },
                issue: {
                    connect: { id: issue.id },
                },
                reason: "Inappropriate",
                status: ReportStatus.Open,
                language: "en",
            },
        });
        testReportIds.push(report.id);

        // Create responses
        // Delete: 200 + 150 = 350 (below MIN_REP.Delete = 500)
        // Hide: 300 (above MIN_REP.HideUntilFixed = 100)
        // NonIssue: 50 (below MIN_REP.NonIssue = 100)
        const responses = await Promise.all([
            DbProvider.get().report_response.create({
                data: {
                    id: generatePK(),
                    report: {
                    connect: { id: report.id },
                },
                createdBy: {
                    connect: { id: voters[0].id },
                },
                    actionSuggested: ReportSuggestedAction.Delete,
                },
            }),
            DbProvider.get().report_response.create({
                data: {
                    id: generatePK(),
                    report: {
                    connect: { id: report.id },
                },
                createdBy: {
                    connect: { id: voters[1].id },
                },
                    actionSuggested: ReportSuggestedAction.Delete,
                },
            }),
            DbProvider.get().report_response.create({
                data: {
                    id: generatePK(),
                    report: {
                    connect: { id: report.id },
                },
                createdBy: {
                    connect: { id: voters[2].id },
                },
                    actionSuggested: ReportSuggestedAction.HideUntilFixed,
                },
            }),
            DbProvider.get().report_response.create({
                data: {
                    id: generatePK(),
                    report: {
                    connect: { id: report.id },
                },
                createdBy: {
                    connect: { id: voters[3].id },
                },
                    actionSuggested: ReportSuggestedAction.NonIssue,
                },
            }),
        ]);
        responses.forEach(r => testReportResponseIds.push(r.id));

        await moderateReports();

        // Should choose Hide since it's the only action meeting minimum
        const updatedReport = await DbProvider.get().report.findUnique({
            where: { id: report.id },
        });
        expect(updatedReport?.status).toBe(ReportStatus.ClosedHidden);

        // Issue can't be hidden (nonHideableTypes), so it should still exist unchanged
        const unchangedIssue = await DbProvider.get().issue.findUnique({
            where: { id: issue.id },
        });
        expect(unchangedIssue).not.toBeNull();
    });

    it("should handle reports with no responses", async () => {
        const reporter = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Lonely Reporter",
                handle: "lonelyreporter",
            },
        });
        testUserIds.push(reporter.id);

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Owner",
                handle: "owner5",
            },
        });
        testUserIds.push(owner.id);

        const tag = await DbProvider.get().tag.create({
            data: {
                id: generatePK(),
                createdBy: {
                    connect: { id: owner.id },
                },
                tag: "lonely-tag",
            },
        });
        testTagIds.push(tag.id);

        // Create report with no responses
        const report = await DbProvider.get().report.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: reporter.id },
                },
                tag: {
                    connect: { id: tag.id },
                },
                reason: "Spam",
                status: ReportStatus.Open,
                language: "en",
            },
        });
        testReportIds.push(report.id);

        await moderateReports();

        // Report should remain open
        const unchangedReport = await DbProvider.get().report.findUnique({
            where: { id: report.id },
        });
        expect(unchangedReport?.status).toBe(ReportStatus.Open);
    });

    it("should handle team-owned objects correctly", async () => {
        const teamOwner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Owner",
                handle: "teamowner",
            },
        });
        testUserIds.push(teamOwner.id);

        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: teamOwner.id },
                },
                handle: "ownerteam",
            },
        });
        testTeamIds.push(team.id);

        const reporter = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Reporter",
                handle: "reporter8",
            },
        });
        testUserIds.push(reporter.id);

        const moderator = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Moderator",
                handle: "moderator2",
                reputation: 600,
            },
        });
        testUserIds.push(moderator.id);

        // Create resource owned by team
        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: teamOwner.id },
                },
                ownedByTeam: {
                    connect: { id: team.id },
                },
                isPrivate: false,
                isInternal: false,
                resourceType: "Routine",
            },
        });
        testRoutineIds.push(resource.id);

        const resourceVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                rootId: resource.id,
                versionLabel: "1.0.0",
                isPrivate: false,
                isDeleted: false,
            },
        });
        testRoutineVersionIds.push(resourceVersion.id);

        // Create report
        const report = await DbProvider.get().report.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: {
                    connect: { id: reporter.id },
                },
                resourceVersion: {

                    connect: { id: resourceVersion.id },

                },
                reason: "Malicious",
                status: ReportStatus.Open,
                language: "en",
            },
        });
        testReportIds.push(report.id);

        // Create response
        const response = await DbProvider.get().report_response.create({
            data: {
                id: generatePK(),
                report: {
                    connect: { id: report.id },
                },
                createdBy: {
                    connect: { id: moderator.id },
                },
                actionSuggested: ReportSuggestedAction.Delete,
            },
        });
        testReportResponseIds.push(response.id);

        await moderateReports();

        // Check that report was closed
        const updatedReport = await DbProvider.get().report.findUnique({
            where: { id: report.id },
        });
        expect(updatedReport?.status).toBe(ReportStatus.ClosedDeleted);

        // Check that resource version was soft-deleted
        const updatedVersion = await DbProvider.get().resource_version.findUnique({
            where: { id: resourceVersion.id },
        });
        expect(updatedVersion?.isDeleted).toBe(true);

        // Verify trigger was called with team owner
        const { Trigger } = await import("@vrooli/server");
        expect(Trigger).toHaveBeenCalled();
        
        // The Trigger mock returns an object with reportActivity method
        const mockTriggerInstance = vi.mocked(Trigger).mock.results[0]?.value;
        expect(mockTriggerInstance?.reportActivity).toHaveBeenCalledWith(
            expect.objectContaining({
                objectOwner: expect.objectContaining({
                    __typename: "Team",
                    id: team.id.toString(),
                }),
            }),
        );
    });
});
