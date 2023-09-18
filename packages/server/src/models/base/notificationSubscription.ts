import { MaxObjects, NotificationSubscriptionSortBy, notificationSubscriptionValidation, SubscribableObject } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull } from "../../builders";
import { defaultPermissions } from "../../utils";
import { NotificationSubscriptionFormat } from "../formats";
import { ModelLogic } from "../types";
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
import { ScheduleModel } from "./schedule";
import { SmartContractModel } from "./smartContract";
import { StandardModel } from "./standard";
import { ApiModelLogic, CommentModelLogic, IssueModelLogic, MeetingModelLogic, NoteModelLogic, NotificationSubscriptionModelLogic, OrganizationModelLogic, ProjectModelLogic, PullRequestModelLogic, QuestionModelLogic, QuizModelLogic, ReportModelLogic, RoutineModelLogic, ScheduleModelLogic, SmartContractModelLogic, StandardModelLogic } from "./types";

export const subscribableMapper: { [key in SubscribableObject]: keyof Prisma.notification_subscriptionUpsertArgs["create"] } = {
    Api: "api",
    Comment: "comment",
    Issue: "issue",
    Meeting: "meeting",
    Note: "note",
    Organization: "organization",
    Project: "project",
    PullRequest: "pullRequest",
    Question: "question",
    Quiz: "quiz",
    Report: "report",
    Routine: "routine",
    Schedule: "schedule",
    SmartContract: "smartContract",
    Standard: "standard",
};

const __typename = "NotificationSubscription" as const;
const suppFields = [] as const;
export const NotificationSubscriptionModel: ModelLogic<NotificationSubscriptionModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.notification_subscription,
    display: {
        label: {
            select: () => ({
                id: true,
                api: { select: ApiModel.display.label.select() },
                comment: { select: CommentModel.display.label.select() },
                issue: { select: IssueModel.display.label.select() },
                meeting: { select: MeetingModel.display.label.select() },
                note: { select: NoteModel.display.label.select() },
                organization: { select: OrganizationModel.display.label.select() },
                project: { select: ProjectModel.display.label.select() },
                pullRequest: { select: PullRequestModel.display.label.select() },
                question: { select: QuestionModel.display.label.select() },
                quiz: { select: QuizModel.display.label.select() },
                report: { select: ReportModel.display.label.select() },
                routine: { select: RoutineModel.display.label.select() },
                schedule: { select: ScheduleModel.display.label.select() },
                smartContract: { select: SmartContractModel.display.label.select() },
                standard: { select: StandardModel.display.label.select() },
            }),
            // Label is first relation that is not null
            get: (select, languages) => {
                if (select.api) return ApiModel.display.label.get(select.api as ApiModelLogic["PrismaModel"], languages);
                if (select.comment) return CommentModel.display.label.get(select.comment as CommentModelLogic["PrismaModel"], languages);
                if (select.issue) return IssueModel.display.label.get(select.issue as IssueModelLogic["PrismaModel"], languages);
                if (select.meeting) return MeetingModel.display.label.get(select.meeting as MeetingModelLogic["PrismaModel"], languages);
                if (select.note) return NoteModel.display.label.get(select.note as NoteModelLogic["PrismaModel"], languages);
                if (select.organization) return OrganizationModel.display.label.get(select.organization as OrganizationModelLogic["PrismaModel"], languages);
                if (select.project) return ProjectModel.display.label.get(select.project as ProjectModelLogic["PrismaModel"], languages);
                if (select.pullRequest) return PullRequestModel.display.label.get(select.pullRequest as PullRequestModelLogic["PrismaModel"], languages);
                if (select.question) return QuestionModel.display.label.get(select.question as QuestionModelLogic["PrismaModel"], languages);
                if (select.quiz) return QuizModel.display.label.get(select.quiz as QuizModelLogic["PrismaModel"], languages);
                if (select.report) return ReportModel.display.label.get(select.report as ReportModelLogic["PrismaModel"], languages);
                if (select.routine) return RoutineModel.display.label.get(select.routine as RoutineModelLogic["PrismaModel"], languages);
                if (select.schedule) return ScheduleModel.display.label.get(select.schedule as ScheduleModelLogic["PrismaModel"], languages);
                if (select.smartContract) return SmartContractModel.display.label.get(select.smartContract as SmartContractModelLogic["PrismaModel"], languages);
                if (select.standard) return StandardModel.display.label.get(select.standard as StandardModelLogic["PrismaModel"], languages);
                return "";
            },
        },
    },
    format: NotificationSubscriptionFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                silent: noNull(data.silent),
                subscriber: { connect: { id: rest.userData.id } },
                [subscribableMapper[data.objectType]]: { connect: { id: data.objectConnect } },
            }),
            update: async ({ data }) => ({
                silent: noNull(data.silent),
            }),
        },
        yup: notificationSubscriptionValidation,
    },
    search: {
        defaultSort: NotificationSubscriptionSortBy.DateCreatedDesc,
        sortBy: NotificationSubscriptionSortBy,
        searchFields: {
            createdTimeFrame: true,
            silent: true,
            objectType: true,
            objectId: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "descriptionWrapped",
                "titleWrapped",
                { api: ApiModel.search.searchStringQuery() },
                { comment: CommentModel.search.searchStringQuery() },
                { issue: IssueModel.search.searchStringQuery() },
                { meeting: MeetingModel.search.searchStringQuery() },
                { note: NoteModel.search.searchStringQuery() },
                { organization: OrganizationModel.search.searchStringQuery() },
                { project: ProjectModel.search.searchStringQuery() },
                { pullRequest: PullRequestModel.search.searchStringQuery() },
                { question: QuestionModel.search.searchStringQuery() },
                { quiz: QuizModel.search.searchStringQuery() },
                { report: ReportModel.search.searchStringQuery() },
                { routine: RoutineModel.search.searchStringQuery() },
                { schedule: ScheduleModel.search.searchStringQuery() },
                { smartContract: SmartContractModel.search.searchStringQuery() },
                { standard: StandardModel.search.searchStringQuery() },
            ],
        }),
        /**
         * Extra protection to ensure only you can see your own subscriptions
         */
        customQueryData: (_, user) => ({ subscriber: { id: user!.id } }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data.subscriber,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            subscriber: "User",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                subscriber: { id: userId },
            }),
        },
    },
});
