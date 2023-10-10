import { MaxObjects, NotificationSubscriptionSortBy, notificationSubscriptionValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { subscribableMapper } from "../../events/subscriber";
import { defaultPermissions } from "../../utils";
import { NotificationSubscriptionFormat } from "../formats";
import { ApiModelInfo, ApiModelLogic, CommentModelInfo, CommentModelLogic, IssueModelInfo, IssueModelLogic, MeetingModelInfo, MeetingModelLogic, NoteModelInfo, NoteModelLogic, NotificationSubscriptionModelLogic, OrganizationModelInfo, OrganizationModelLogic, ProjectModelInfo, ProjectModelLogic, PullRequestModelInfo, PullRequestModelLogic, QuestionModelInfo, QuestionModelLogic, QuizModelInfo, QuizModelLogic, ReportModelInfo, ReportModelLogic, RoutineModelInfo, RoutineModelLogic, ScheduleModelInfo, ScheduleModelLogic, SmartContractModelInfo, SmartContractModelLogic, StandardModelInfo, StandardModelLogic } from "./types";

const __typename = "NotificationSubscription" as const;
export const NotificationSubscriptionModel: NotificationSubscriptionModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.notification_subscription,
    display: {
        label: {
            select: () => ({
                id: true,
                api: { select: ModelMap.get<ApiModelLogic>("Api").display.label.select() },
                comment: { select: ModelMap.get<CommentModelLogic>("Comment").display.label.select() },
                issue: { select: ModelMap.get<IssueModelLogic>("Issue").display.label.select() },
                meeting: { select: ModelMap.get<MeetingModelLogic>("Meeting").display.label.select() },
                note: { select: ModelMap.get<NoteModelLogic>("Note").display.label.select() },
                organization: { select: ModelMap.get<OrganizationModelLogic>("Organization").display.label.select() },
                project: { select: ModelMap.get<ProjectModelLogic>("Project").display.label.select() },
                pullRequest: { select: ModelMap.get<PullRequestModelLogic>("PullRequest").display.label.select() },
                question: { select: ModelMap.get<QuestionModelLogic>("Question").display.label.select() },
                quiz: { select: ModelMap.get<QuizModelLogic>("Quiz").display.label.select() },
                report: { select: ModelMap.get<ReportModelLogic>("Report").display.label.select() },
                routine: { select: ModelMap.get<RoutineModelLogic>("Routine").display.label.select() },
                schedule: { select: ModelMap.get<ScheduleModelLogic>("Schedule").display.label.select() },
                smartContract: { select: ModelMap.get<SmartContractModelLogic>("SmartContract").display.label.select() },
                standard: { select: ModelMap.get<StandardModelLogic>("Standard").display.label.select() },
            }),
            // Label is first relation that is not null
            get: (select, languages) => {
                if (select.api) return ModelMap.get<ApiModelLogic>("Api").display.label.get(select.api as ApiModelInfo["PrismaModel"], languages);
                if (select.comment) return ModelMap.get<CommentModelLogic>("Comment").display.label.get(select.comment as CommentModelInfo["PrismaModel"], languages);
                if (select.issue) return ModelMap.get<IssueModelLogic>("Issue").display.label.get(select.issue as IssueModelInfo["PrismaModel"], languages);
                if (select.meeting) return ModelMap.get<MeetingModelLogic>("Meeting").display.label.get(select.meeting as MeetingModelInfo["PrismaModel"], languages);
                if (select.note) return ModelMap.get<NoteModelLogic>("Note").display.label.get(select.note as NoteModelInfo["PrismaModel"], languages);
                if (select.organization) return ModelMap.get<OrganizationModelLogic>("Organization").display.label.get(select.organization as OrganizationModelInfo["PrismaModel"], languages);
                if (select.project) return ModelMap.get<ProjectModelLogic>("Project").display.label.get(select.project as ProjectModelInfo["PrismaModel"], languages);
                if (select.pullRequest) return ModelMap.get<PullRequestModelLogic>("PullRequest").display.label.get(select.pullRequest as PullRequestModelInfo["PrismaModel"], languages);
                if (select.question) return ModelMap.get<QuestionModelLogic>("Question").display.label.get(select.question as QuestionModelInfo["PrismaModel"], languages);
                if (select.quiz) return ModelMap.get<QuizModelLogic>("Quiz").display.label.get(select.quiz as QuizModelInfo["PrismaModel"], languages);
                if (select.report) return ModelMap.get<ReportModelLogic>("Report").display.label.get(select.report as ReportModelInfo["PrismaModel"], languages);
                if (select.routine) return ModelMap.get<RoutineModelLogic>("Routine").display.label.get(select.routine as RoutineModelInfo["PrismaModel"], languages);
                if (select.schedule) return ModelMap.get<ScheduleModelLogic>("Schedule").display.label.get(select.schedule as ScheduleModelInfo["PrismaModel"], languages);
                if (select.smartContract) return ModelMap.get<SmartContractModelLogic>("SmartContract").display.label.get(select.smartContract as SmartContractModelInfo["PrismaModel"], languages);
                if (select.standard) return ModelMap.get<StandardModelLogic>("Standard").display.label.get(select.standard as StandardModelInfo["PrismaModel"], languages);
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
                { api: ModelMap.get<ApiModelLogic>("Api").search.searchStringQuery() },
                { comment: ModelMap.get<CommentModelLogic>("Comment").search.searchStringQuery() },
                { issue: ModelMap.get<IssueModelLogic>("Issue").search.searchStringQuery() },
                { meeting: ModelMap.get<MeetingModelLogic>("Meeting").search.searchStringQuery() },
                { note: ModelMap.get<NoteModelLogic>("Note").search.searchStringQuery() },
                { organization: ModelMap.get<OrganizationModelLogic>("Organization").search.searchStringQuery() },
                { project: ModelMap.get<ProjectModelLogic>("Project").search.searchStringQuery() },
                { pullRequest: ModelMap.get<PullRequestModelLogic>("PullRequest").search.searchStringQuery() },
                { question: ModelMap.get<QuestionModelLogic>("Question").search.searchStringQuery() },
                { quiz: ModelMap.get<QuizModelLogic>("Quiz").search.searchStringQuery() },
                { report: ModelMap.get<ReportModelLogic>("Report").search.searchStringQuery() },
                { routine: ModelMap.get<RoutineModelLogic>("Routine").search.searchStringQuery() },
                { schedule: ModelMap.get<ScheduleModelLogic>("Schedule").search.searchStringQuery() },
                { smartContract: ModelMap.get<SmartContractModelLogic>("SmartContract").search.searchStringQuery() },
                { standard: ModelMap.get<StandardModelLogic>("Standard").search.searchStringQuery() },
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
            User: data?.subscriber,
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
