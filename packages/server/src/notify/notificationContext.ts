import { logger } from "../events";

export type ApiSubscriptionContext = object;

export type CommentSubscriptionContext = object

export type IssueSubscriptionContext = object

export type MeetingSubscriptionContext = object

export type NoteSubscriptionContext = object

export type OrganizationSubscriptionContext = object

export type ProjectSubscriptionContext = object

export type PullRequestSubscriptionContext = object

export type QuestionSubscriptionContext = object

export type QuizSubscriptionContext = object

export type ReportSubscriptionContext = object

export type RoutineSubscriptionContext = object

export type ScheduleSubscriptionContext = {
    reminders: {
        email: boolean,
        phone: boolean,
        push: boolean,
        minutesBefore: number,
    }[];
}

export type SmartContractSubscriptionContext = object

export type StandupSubscriptionContext = object

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
export const parseSubscriptionContext = (contextJson: string | null): SubscriptionContext | object => {
    try {
        const settings = contextJson ? JSON.parse(contextJson) : {};
        return settings;
    } catch (error) {
        logger.error("Failed to parse notification subscription context", { trace: "0431" });
        return {};
    }
};
