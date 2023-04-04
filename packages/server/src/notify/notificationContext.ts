import { logger } from "../events";

export type ApiSubscriptionContext = {}

export type CommentSubscriptionContext = {}

export type IssueSubscriptionContext = {}

export type MeetingSubscriptionContext = {}

export type NoteSubscriptionContext = {}

export type OrganizationSubscriptionContext = {}

export type ProjectSubscriptionContext = {}

export type PullRequestSubscriptionContext = {}

export type QuestionSubscriptionContext = {}

export type QuizSubscriptionContext = {}

export type ReportSubscriptionContext = {}

export type RoutineSubscriptionContext = {}

export type ScheduleSubscriptionContext = {
    reminders: {
        email: boolean,
        phone: boolean,
        push: boolean,
        minutesBefore: number,
    }[];
}

export type SmartContractSubscriptionContext = {}

export type StandupSubscriptionContext = {}

export type SubscriptionContext = ApiSubscriptionContext |
    CommentSubscriptionContext |
    IssueSubscriptionContext |
    MeetingSubscriptionContext |
    NoteSubscriptionContext |
    OrganizationSubscriptionContext |
    ProjectSubscriptionContext |
    PullRequestSubscriptionContext |
    QuestionSubscriptionContext |
    QuizSubscriptionContext |
    ReportSubscriptionContext |
    RoutineSubscriptionContext |
    ScheduleSubscriptionContext |
    SmartContractSubscriptionContext |
    StandupSubscriptionContext;

/**
 * Parses a notification subscription's context from its stringified JSON representation
 * @param contextJson JSON string of the subscription's context
 * @returns Parsed subscription context, or empty object if there is an error
 */
export const parseSubscriptionContext = (contextJson: string | null): SubscriptionContext | {} => {
    try {
        const settings = contextJson ? JSON.parse(contextJson) : {};
        return settings;
    } catch (error) {
        logger.error(`Failed to parse notification subscription context`, { trace: '0431' });
        return {}
    }
}