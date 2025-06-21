/**
 * Authoritative mapping of Prisma model names to their corresponding type names
 * This file is auto-generated based on the Prisma schema
 * 
 * Usage:
 * ```typescript
 * import { PRISMA_TYPE_MAP } from './PRISMA_TYPE_MAP.js';
 * const userTypes = PRISMA_TYPE_MAP['user'];
 * // userTypes.createInput === 'userCreateInput'
 * ```
 */

export const PRISMA_TYPE_MAP = {
    // Core models
    'user': {
        model: 'user',
        createInput: 'userCreateInput',
        updateInput: 'userUpdateInput',
        include: 'userInclude'
    },
    'team': {
        model: 'team',
        createInput: 'teamCreateInput',
        updateInput: 'teamUpdateInput',
        include: 'teamInclude'
    },
    'award': {
        model: 'award',
        createInput: 'awardCreateInput',
        updateInput: 'awardUpdateInput',
        include: 'awardInclude'
    },
    'api_key': {
        model: 'api_key',
        createInput: 'api_keyCreateInput',
        updateInput: 'api_keyUpdateInput',
        include: 'api_keyInclude'
    },
    'api_key_external': {
        model: 'api_key_external',
        createInput: 'api_key_externalCreateInput',
        updateInput: 'api_key_externalUpdateInput',
        include: 'api_key_externalInclude'
    },
    'bookmark': {
        model: 'bookmark',
        createInput: 'bookmarkCreateInput',
        updateInput: 'bookmarkUpdateInput',
        include: 'bookmarkInclude'
    },
    'bookmark_list': {
        model: 'bookmark_list',
        createInput: 'bookmark_listCreateInput',
        updateInput: 'bookmark_listUpdateInput',
        include: 'bookmark_listInclude'
    },
    'chat': {
        model: 'chat',
        createInput: 'chatCreateInput',
        updateInput: 'chatUpdateInput',
        include: 'chatInclude'
    },
    'chat_translation': {
        model: 'chat_translation',
        createInput: 'chat_translationCreateInput',
        updateInput: 'chat_translationUpdateInput',
        include: 'chat_translationInclude'
    },
    'chat_message': {
        model: 'chat_message',
        createInput: 'chat_messageCreateInput',
        updateInput: 'chat_messageUpdateInput',
        include: 'chat_messageInclude'
    },
    'chat_participants': {
        model: 'chat_participants',
        createInput: 'chat_participantsCreateInput',
        updateInput: 'chat_participantsUpdateInput',
        include: 'chat_participantsInclude'
    },
    'chat_invite': {
        model: 'chat_invite',
        createInput: 'chat_inviteCreateInput',
        updateInput: 'chat_inviteUpdateInput',
        include: 'chat_inviteInclude'
    },
    'comment': {
        model: 'comment',
        createInput: 'commentCreateInput',
        updateInput: 'commentUpdateInput',
        include: 'commentInclude'
    },
    'comment_translation': {
        model: 'comment_translation',
        createInput: 'comment_translationCreateInput',
        updateInput: 'comment_translationUpdateInput',
        include: 'comment_translationInclude'
    },
    'credit_account': {
        model: 'credit_account',
        createInput: 'credit_accountCreateInput',
        updateInput: 'credit_accountUpdateInput',
        include: 'credit_accountInclude'
    },
    'credit_ledger_entry': {
        model: 'credit_ledger_entry',
        createInput: 'credit_ledger_entryCreateInput',
        updateInput: 'credit_ledger_entryUpdateInput',
        include: 'credit_ledger_entryInclude'
    },
    'email': {
        model: 'email',
        createInput: 'emailCreateInput',
        updateInput: 'emailUpdateInput',
        include: 'emailInclude'
    },
    'issue': {
        model: 'issue',
        createInput: 'issueCreateInput',
        updateInput: 'issueUpdateInput',
        include: 'issueInclude'
    },
    'issue_translation': {
        model: 'issue_translation',
        createInput: 'issue_translationCreateInput',
        updateInput: 'issue_translationUpdateInput',
        include: 'issue_translationInclude'
    },
    'meeting': {
        model: 'meeting',
        createInput: 'meetingCreateInput',
        updateInput: 'meetingUpdateInput',
        include: 'meetingInclude'
    },
    'meeting_attendees': {
        model: 'meeting_attendees',
        createInput: 'meeting_attendeesCreateInput',
        updateInput: 'meeting_attendeesUpdateInput',
        include: 'meeting_attendeesInclude'
    },
    'meeting_invite': {
        model: 'meeting_invite',
        createInput: 'meeting_inviteCreateInput',
        updateInput: 'meeting_inviteUpdateInput',
        include: 'meeting_inviteInclude'
    },
    'meeting_translation': {
        model: 'meeting_translation',
        createInput: 'meeting_translationCreateInput',
        updateInput: 'meeting_translationUpdateInput',
        include: 'meeting_translationInclude'
    },
    'member': {
        model: 'member',
        createInput: 'memberCreateInput',
        updateInput: 'memberUpdateInput',
        include: 'memberInclude'
    },
    'member_invite': {
        model: 'member_invite',
        createInput: 'member_inviteCreateInput',
        updateInput: 'member_inviteUpdateInput',
        include: 'member_inviteInclude'
    },
    'notification': {
        model: 'notification',
        createInput: 'notificationCreateInput',
        updateInput: 'notificationUpdateInput',
        include: 'notificationInclude'
    },
    'notification_subscription': {
        model: 'notification_subscription',
        createInput: 'notification_subscriptionCreateInput',
        updateInput: 'notification_subscriptionUpdateInput',
        include: 'notification_subscriptionInclude'
    },
    'payment': {
        model: 'payment',
        createInput: 'paymentCreateInput',
        updateInput: 'paymentUpdateInput',
        include: 'paymentInclude'
    },
    'phone': {
        model: 'phone',
        createInput: 'phoneCreateInput',
        updateInput: 'phoneUpdateInput',
        include: 'phoneInclude'
    },
    'plan': {
        model: 'plan',
        createInput: 'planCreateInput',
        updateInput: 'planUpdateInput',
        include: 'planInclude'
    },
    'pull_request': {
        model: 'pull_request',
        createInput: 'pull_requestCreateInput',
        updateInput: 'pull_requestUpdateInput',
        include: 'pull_requestInclude'
    },
    'pull_request_translation': {
        model: 'pull_request_translation',
        createInput: 'pull_request_translationCreateInput',
        updateInput: 'pull_request_translationUpdateInput',
        include: 'pull_request_translationInclude'
    },
    'push_device': {
        model: 'push_device',
        createInput: 'push_deviceCreateInput',
        updateInput: 'push_deviceUpdateInput',
        include: 'push_deviceInclude'
    },
    'reaction': {
        model: 'reaction',
        createInput: 'reactionCreateInput',
        updateInput: 'reactionUpdateInput',
        include: 'reactionInclude'
    },
    'reaction_summary': {
        model: 'reaction_summary',
        createInput: 'reaction_summaryCreateInput',
        updateInput: 'reaction_summaryUpdateInput',
        include: 'reaction_summaryInclude'
    },
    'reminder': {
        model: 'reminder',
        createInput: 'reminderCreateInput',
        updateInput: 'reminderUpdateInput',
        include: 'reminderInclude'
    },
    'reminder_item': {
        model: 'reminder_item',
        createInput: 'reminder_itemCreateInput',
        updateInput: 'reminder_itemUpdateInput',
        include: 'reminder_itemInclude'
    },
    'reminder_list': {
        model: 'reminder_list',
        createInput: 'reminder_listCreateInput',
        updateInput: 'reminder_listUpdateInput',
        include: 'reminder_listInclude'
    },
    'report': {
        model: 'report',
        createInput: 'reportCreateInput',
        updateInput: 'reportUpdateInput',
        include: 'reportInclude'
    },
    'report_response': {
        model: 'report_response',
        createInput: 'report_responseCreateInput',
        updateInput: 'report_responseUpdateInput',
        include: 'report_responseInclude'
    },
    'reputation_history': {
        model: 'reputation_history',
        createInput: 'reputation_historyCreateInput',
        updateInput: 'reputation_historyUpdateInput',
        include: 'reputation_historyInclude'
    },
    'resource': {
        model: 'resource',
        createInput: 'resourceCreateInput',
        updateInput: 'resourceUpdateInput',
        include: 'resourceInclude'
    },
    'resource_version': {
        model: 'resource_version',
        createInput: 'resource_versionCreateInput',
        updateInput: 'resource_versionUpdateInput',
        include: 'resource_versionInclude'
    },
    'resource_version_relation': {
        model: 'resource_version_relation',
        createInput: 'resource_version_relationCreateInput',
        updateInput: 'resource_version_relationUpdateInput',
        include: 'resource_version_relationInclude'
    },
    'resource_translation': {
        model: 'resource_translation',
        createInput: 'resource_translationCreateInput',
        updateInput: 'resource_translationUpdateInput',
        include: 'resource_translationInclude'
    },
    'resource_tag': {
        model: 'resource_tag',
        createInput: 'resource_tagCreateInput',
        updateInput: 'resource_tagUpdateInput',
        include: 'resource_tagInclude'
    },
    'run': {
        model: 'run',
        createInput: 'runCreateInput',
        updateInput: 'runUpdateInput',
        include: 'runInclude'
    },
    'run_io': {
        model: 'run_io',
        createInput: 'run_ioCreateInput',
        updateInput: 'run_ioUpdateInput',
        include: 'run_ioInclude'
    },
    'run_step': {
        model: 'run_step',
        createInput: 'run_stepCreateInput',
        updateInput: 'run_stepUpdateInput',
        include: 'run_stepInclude'
    },
    'schedule': {
        model: 'schedule',
        createInput: 'scheduleCreateInput',
        updateInput: 'scheduleUpdateInput',
        include: 'scheduleInclude'
    },
    'schedule_exception': {
        model: 'schedule_exception',
        createInput: 'schedule_exceptionCreateInput',
        updateInput: 'schedule_exceptionUpdateInput',
        include: 'schedule_exceptionInclude'
    },
    'schedule_recurrence': {
        model: 'schedule_recurrence',
        createInput: 'schedule_recurrenceCreateInput',
        updateInput: 'schedule_recurrenceUpdateInput',
        include: 'schedule_recurrenceInclude'
    },
    'session': {
        model: 'session',
        createInput: 'sessionCreateInput',
        updateInput: 'sessionUpdateInput',
        include: 'sessionInclude'
    },
    'stats_resource': {
        model: 'stats_resource',
        createInput: 'stats_resourceCreateInput',
        updateInput: 'stats_resourceUpdateInput',
        include: 'stats_resourceInclude'
    },
    'stats_site': {
        model: 'stats_site',
        createInput: 'stats_siteCreateInput',
        updateInput: 'stats_siteUpdateInput',
        include: 'stats_siteInclude'
    },
    'stats_team': {
        model: 'stats_team',
        createInput: 'stats_teamCreateInput',
        updateInput: 'stats_teamUpdateInput',
        include: 'stats_teamInclude'
    },
    'stats_user': {
        model: 'stats_user',
        createInput: 'stats_userCreateInput',
        updateInput: 'stats_userUpdateInput',
        include: 'stats_userInclude'
    },
    'tag': {
        model: 'tag',
        createInput: 'tagCreateInput',
        updateInput: 'tagUpdateInput',
        include: 'tagInclude'
    },
    'tag_translation': {
        model: 'tag_translation',
        createInput: 'tag_translationCreateInput',
        updateInput: 'tag_translationUpdateInput',
        include: 'tag_translationInclude'
    },
    'team_tag': {
        model: 'team_tag',
        createInput: 'team_tagCreateInput',
        updateInput: 'team_tagUpdateInput',
        include: 'team_tagInclude'
    },
    'team_translation': {
        model: 'team_translation',
        createInput: 'team_translationCreateInput',
        updateInput: 'team_translationUpdateInput',
        include: 'team_translationInclude'
    },
    'transfer': {
        model: 'transfer',
        createInput: 'transferCreateInput',
        updateInput: 'transferUpdateInput',
        include: 'transferInclude'
    },
    'user_auth': {
        model: 'user_auth',
        createInput: 'user_authCreateInput',
        updateInput: 'user_authUpdateInput',
        include: 'user_authInclude'
    },
    'user_translation': {
        model: 'user_translation',
        createInput: 'user_translationCreateInput',
        updateInput: 'user_translationUpdateInput',
        include: 'user_translationInclude'
    },
    'view': {
        model: 'view',
        createInput: 'viewCreateInput',
        updateInput: 'viewUpdateInput',
        include: 'viewInclude'
    },
    'wallet': {
        model: 'wallet',
        createInput: 'walletCreateInput',
        updateInput: 'walletUpdateInput',
        include: 'walletInclude'
    }
} as const;

// Helper type to get Prisma types for a model
export type PrismaModelTypes<T extends keyof typeof PRISMA_TYPE_MAP> = typeof PRISMA_TYPE_MAP[T];