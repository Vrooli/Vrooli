/**
 * Test cleanup helpers with dependency-ordered table cleanup
 * 
 * These helpers ensure proper cleanup order based on foreign key constraints
 * to prevent constraint violations during test cleanup.
 * 
 * CRITICAL: Tables must be deleted in child-first order due to foreign key constraints.
 * If you add new tables, place them in the correct dependency order.
 */

import { type PrismaClient } from "@prisma/client";

/**
 * Comprehensive cleanup for tests that create complex data structures
 * Cleans all tables in proper dependency order
 */
export async function cleanupFull(prisma: PrismaClient): Promise<void> {
    // Children first, parents last - CRITICAL for FK constraints!
    
    // Level 1: Deepest children (no dependencies on other app tables)
    await prisma.award.deleteMany();
    await prisma.user_translation.deleteMany();
    await prisma.team_translation.deleteMany();
    await prisma.chat_translation.deleteMany();
    await prisma.comment_translation.deleteMany();
    await prisma.issue_translation.deleteMany();
    await prisma.meeting_translation.deleteMany();
    await prisma.pull_request_translation.deleteMany();
    await prisma.resource_translation.deleteMany();
    
    // Level 2: Run execution hierarchy (deepest execution children)
    await prisma.run_step.deleteMany();
    await prisma.run_io.deleteMany();
    
    // Level 3: Chat hierarchy children
    await prisma.chat_message.deleteMany();
    await prisma.chat_participants.deleteMany();
    await prisma.chat_invite.deleteMany();
    
    // Level 4: Meeting children
    await prisma.meeting_attendees.deleteMany();
    await prisma.meeting_invite.deleteMany();
    
    // Level 5: Team membership and invites
    await prisma.member.deleteMany();
    await prisma.member_invite.deleteMany();
    
    // Level 6: Authentication and session children
    await prisma.session.deleteMany();
    await prisma.user_auth.deleteMany();
    
    // Level 7: Contact information
    await prisma.email.deleteMany();
    await prisma.phone.deleteMany();
    await prisma.push_device.deleteMany();
    
    // Level 8: Financial and credit system
    await prisma.credit_ledger_entry.deleteMany();
    await prisma.credit_account.deleteMany();
    await prisma.payment.deleteMany();
    await prisma.plan.deleteMany();
    
    // Level 9: Resource relationships and comments
    await prisma.resource_version_relation.deleteMany();
    await prisma.comment.deleteMany();
    await prisma.pull_request.deleteMany();
    
    // Level 10: Bookmarks, reactions, notifications
    await prisma.bookmark.deleteMany();
    await prisma.bookmark_list.deleteMany();
    await prisma.reaction.deleteMany();
    await prisma.reaction_summary.deleteMany();
    await prisma.notification.deleteMany();
    await prisma.notification_subscription.deleteMany();
    
    // Level 11: API keys and external integrations
    await prisma.api_key.deleteMany();
    await prisma.api_key_external.deleteMany();
    
    // Level 12: Issues and reports
    await prisma.issue.deleteMany();
    await prisma.report.deleteMany();
    await prisma.report_response.deleteMany();
    
    // Level 13: Resource hierarchy
    await prisma.resource_version.deleteMany();
    await prisma.run.deleteMany();
    
    // Level 14: Schedules and reminders
    await prisma.schedule_exception.deleteMany();
    await prisma.schedule_recurrence.deleteMany();
    await prisma.schedule.deleteMany();
    await prisma.reminder_item.deleteMany();
    await prisma.reminder_list.deleteMany();
    await prisma.reminder.deleteMany();
    
    // Level 15: Views and stats
    await prisma.view.deleteMany();
    await prisma.stats_site.deleteMany();
    await prisma.stats_team.deleteMany();
    await prisma.stats_user.deleteMany();
    await prisma.stats_resource.deleteMany();
    
    // Level 16: Tags and transfers
    await prisma.resource_tag.deleteMany();
    await prisma.team_tag.deleteMany();
    await prisma.tag.deleteMany();
    await prisma.transfer.deleteMany();
    await prisma.wallet.deleteMany();
    
    // Level 17: Main entity hierarchy (parents)
    await prisma.meeting.deleteMany();
    await prisma.chat.deleteMany();
    await prisma.resource.deleteMany();
    await prisma.team.deleteMany();
    await prisma.user.deleteMany();
}

/**
 * User authentication cleanup - for auth-related tests
 */
export async function cleanupUserAuth(prisma: PrismaClient): Promise<void> {
    await prisma.session.deleteMany();
    await prisma.user_auth.deleteMany();
    await prisma.email.deleteMany();
    await prisma.phone.deleteMany();
    await prisma.push_device.deleteMany();
    await prisma.user_translation.deleteMany();
    await prisma.plan.deleteMany();
    await prisma.user.deleteMany();
}

/**
 * Chat system cleanup - for chat-related tests  
 */
export async function cleanupChat(prisma: PrismaClient): Promise<void> {
    await prisma.chat_message.deleteMany();
    await prisma.chat_participants.deleteMany();
    await prisma.chat_invite.deleteMany();
    await prisma.chat_translation.deleteMany();
    await prisma.notification_subscription.deleteMany();
    await prisma.chat.deleteMany();
    await prisma.user.deleteMany();
}

/**
 * Team system cleanup - for team-related tests
 */
export async function cleanupTeam(prisma: PrismaClient): Promise<void> {
    await prisma.member.deleteMany();
    await prisma.member_invite.deleteMany();
    await prisma.meeting_attendees.deleteMany();
    await prisma.meeting_invite.deleteMany();
    await prisma.meeting_translation.deleteMany();
    await prisma.meeting.deleteMany();
    await prisma.team_translation.deleteMany();
    await prisma.team_tag.deleteMany();
    await prisma.api_key.deleteMany();
    await prisma.email.deleteMany();
    await prisma.phone.deleteMany();
    await prisma.credit_account.deleteMany();
    await prisma.plan.deleteMany();
    await prisma.team.deleteMany();
    await prisma.user.deleteMany();
}

/**
 * Resource and execution cleanup - for resource/run tests
 */
export async function cleanupExecution(prisma: PrismaClient): Promise<void> {
    await prisma.run_step.deleteMany();
    await prisma.run_io.deleteMany();
    await prisma.run.deleteMany();
    await prisma.resource_version_relation.deleteMany();
    await prisma.resource_version.deleteMany();
    await prisma.resource_translation.deleteMany();
    await prisma.resource_tag.deleteMany();
    await prisma.schedule_exception.deleteMany();
    await prisma.schedule_recurrence.deleteMany();
    await prisma.schedule.deleteMany();
    await prisma.resource.deleteMany();
    await prisma.tag.deleteMany();
    await prisma.user.deleteMany();
}

/**
 * Financial system cleanup - for payment/credit tests
 */
export async function cleanupFinancial(prisma: PrismaClient): Promise<void> {
    await prisma.credit_ledger_entry.deleteMany();
    await prisma.credit_account.deleteMany();
    await prisma.payment.deleteMany();
    await prisma.plan.deleteMany();
    await prisma.transfer.deleteMany();
    await prisma.wallet.deleteMany();
    await prisma.team.deleteMany();
    await prisma.user.deleteMany();
}

/**
 * Content interaction cleanup - for comments, reactions, bookmarks
 */
export async function cleanupContent(prisma: PrismaClient): Promise<void> {
    await prisma.comment_translation.deleteMany();
    await prisma.comment.deleteMany();
    await prisma.reaction.deleteMany();
    await prisma.reaction_summary.deleteMany();
    await prisma.bookmark.deleteMany();
    await prisma.bookmark_list.deleteMany();
    await prisma.view.deleteMany();
    await prisma.notification_subscription.deleteMany();
    await prisma.notification.deleteMany();
    await prisma.issue_translation.deleteMany();
    await prisma.issue.deleteMany();
    await prisma.pull_request_translation.deleteMany();
    await prisma.pull_request.deleteMany();
    await prisma.report_response.deleteMany();
    await prisma.report.deleteMany();
    await prisma.resource.deleteMany();
    await prisma.user.deleteMany();
}

/**
 * Lightweight cleanup for simple tests that only create basic user data
 */
export async function cleanupMinimal(prisma: PrismaClient): Promise<void> {
    await prisma.session.deleteMany();
    await prisma.user_auth.deleteMany();
    await prisma.email.deleteMany();
    await prisma.user.deleteMany();
}

/**
 * All cleanup functions organized by scope
 */
export const cleanupGroups = {
    /** Complete cleanup for complex integration tests */
    full: cleanupFull,
    /** User authentication and basic user data */
    userAuth: cleanupUserAuth,
    /** Chat system and messaging */
    chat: cleanupChat,
    /** Team management and membership */
    team: cleanupTeam,
    /** Resource execution and routines */
    execution: cleanupExecution,
    /** Payment and credit system */
    financial: cleanupFinancial,
    /** Content interactions (comments, reactions, etc.) */
    content: cleanupContent,
    /** Minimal cleanup for simple tests */
    minimal: cleanupMinimal,
} as const;

export type CleanupGroup = keyof typeof cleanupGroups;
