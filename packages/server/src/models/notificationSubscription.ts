import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { NotificationSubscription, NotificationSubscriptionCreateInput, NotificationSubscriptionSearchInput, NotificationSubscriptionSortBy, NotificationSubscriptionUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ApiModel } from "./api";
import { CommentModel } from "./comment";
import { IssueModel } from "./issue";
import { MeetingModel } from "./meeting";
import { NoteModel } from "./note";
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { PullRequestModel } from "./pullRequest";
import { QuestionModel } from "./question";
import { QuizModel } from "./quiz";
import { ReportModel } from "./report";
import { RoutineModel } from "./routine";
import { SmartContractModel } from "./smartContract";
import { StandardModel } from "./standard";
import { ModelLogic } from "./types";

export const subscriberMapper: { [x: string]: string } = {
    Api: 'api',
    Comment: 'comment',
    Issue: 'issue',
    Note: 'note',
    Organization: 'organization',
    Project: 'project',
    PullRequest: 'pullRequest',
    Question: 'question',
    Quiz: 'quiz',
    Report: 'report',
    Routine: 'routine',
    SmartContract: 'smartContract',
    Standard: 'standard',
}

const __typename = 'NotificationSubscription' as const;
const suppFields = [] as const;
export const NotificationSubscriptionModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NotificationSubscriptionCreateInput,
    GqlUpdate: NotificationSubscriptionUpdateInput,
    GqlModel: NotificationSubscription,
    GqlSearch: NotificationSubscriptionSearchInput,
    GqlSort: NotificationSubscriptionSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.notification_subscriptionUpsertArgs['create'],
    PrismaUpdate: Prisma.notification_subscriptionUpsertArgs['update'],
    PrismaModel: Prisma.notification_subscriptionGetPayload<SelectWrap<Prisma.notification_subscriptionSelect>>,
    PrismaSelect: Prisma.notification_subscriptionSelect,
    PrismaWhere: Prisma.notification_subscriptionWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.notification_subscription,
    display: {
        select: () => ({
            id: true,
            api: { select: ApiModel.display.select() },
            comment: { select: CommentModel.display.select() },
            issue: { select: IssueModel.display.select() },
            meeting: { select: MeetingModel.display.select() },
            note: { select: NoteModel.display.select() },
            organization: { select: OrganizationModel.display.select() },
            project: { select: ProjectModel.display.select() },
            pullRequest: { select: PullRequestModel.display.select() },
            question: { select: QuestionModel.display.select() },
            quiz: { select: QuizModel.display.select() },
            report: { select: ReportModel.display.select() },
            routine: { select: RoutineModel.display.select() },
            smartContract: { select: SmartContractModel.display.select() },
            standard: { select: StandardModel.display.select() },
        }),
        // Label is first relation that is not null
        label: (select, languages) => {
            if (select.api) return ApiModel.display.label(select.api as any, languages)
            if (select.comment) return CommentModel.display.label(select.comment as any, languages)
            if (select.issue) return IssueModel.display.label(select.issue as any, languages)
            if (select.meeting) return MeetingModel.display.label(select.meeting as any, languages)
            if (select.note) return NoteModel.display.label(select.note as any, languages)
            if (select.organization) return OrganizationModel.display.label(select.organization as any, languages)
            if (select.project) return ProjectModel.display.label(select.project as any, languages)
            if (select.pullRequest) return PullRequestModel.display.label(select.pullRequest as any, languages)
            if (select.question) return QuestionModel.display.label(select.question as any, languages)
            if (select.quiz) return QuizModel.display.label(select.quiz as any, languages)
            if (select.report) return ReportModel.display.label(select.report as any, languages)
            if (select.routine) return RoutineModel.display.label(select.routine as any, languages)
            if (select.smartContract) return SmartContractModel.display.label(select.smartContract as any, languages)   
            if (select.standard) return SmartContractModel.display.label(select.standard as any, languages)
            return '';
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            object: {
                api: 'Api',
                comment: 'Comment',
                issue: 'Issue',
                meeting: 'Meeting',
                note: 'Note',
                organization: 'Organization',
                project: 'Project',
                pullRequest: 'PullRequest',
                question: 'Question',
                quiz: 'Quiz',
                report: 'Report',
                routine: 'Routine',
                smartContract: 'SmartContract',
                standard: 'Standard',
            },
        },
        prismaRelMap: {
            __typename,
            api: 'Api',
            comment: 'Comment',
            issue: 'Issue',
            meeting: 'Meeting',
            note: 'Note',
            organization: 'Organization',
            project: 'Project',
            pullRequest: 'PullRequest',
            question: 'Question',
            quiz: 'Quiz',
            report: 'Report',
            routine: 'Routine',
            smartContract: 'SmartContract',
            standard: 'Standard',
            subscriber: 'User',
        },
        countFields: {},
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})