import { GqlModelType } from '@shared/consts';
import { ApiModel } from './api';
import { ApiKeyModel } from './apiKey';
import { ApiVersionModel } from './apiVersion';
import { AwardModel } from './award';
import { BookmarkModel } from './bookmark';
import { CommentModel } from './comment';
import { EmailModel } from './email';
import { IssueModel } from './issue';
import { LabelModel } from './label';
import { MeetingModel } from './meeting';
import { MeetingInviteModel } from './meetingInvite';
import { MemberModel } from './member';
import { MemberInviteModel } from './memberInvite';
import { NodeModel } from './node';
import { NodeEndModel } from './nodeEnd';
import { NodeLinkModel } from './nodeLink';
import { NodeLinkWhenModel } from './nodeLinkWhen';
import { NodeLoopModel } from './nodeLoop';
import { NodeLoopWhileModel } from './nodeLoopWhile';
import { NodeRoutineListModel } from './nodeRoutineList';
import { NodeRoutineListItemModel } from './nodeRoutineListItem';
import { NoteModel } from './note';
import { NoteVersionModel } from './noteVersion';
import { NotificationModel } from './notification';
import { NotificationSubscriptionModel } from './notificationSubscription';
import { OrganizationModel } from './organization';
import { PaymentModel } from './payment';
import { PhoneModel } from './phone';
import { PostModel } from './post';
import { PremiumModel } from './premium';
import { ProjectModel } from './project';
import { ProjectVersionModel } from './projectVersion';
import { ProjectVersionDirectoryModel } from './projectVersionDirectory';
import { PullRequestModel } from './pullRequest';
import { PushDeviceModel } from './pushDevice';
import { QuestionModel } from './question';
import { QuestionAnswerModel } from './questionAnswer';
import { QuizModel } from './quiz';
import { QuizAttemptModel } from './quizAttempt';
import { QuizQuestionModel } from './quizQuestion';
import { QuizQuestionResponseModel } from './quizQuestionResponse';
import { ReminderModel } from './reminder';
import { ReminderItemModel } from './reminderItem';
import { ReminderListModel } from './reminderList';
import { ReportModel } from './report';
import { ResourceModel } from './resource';
import { ResourceListModel } from './resourceList';
import { RoleModel } from './role';
import { RoutineModel } from './routine';
import { RoutineVersionModel } from './routineVersion';
import { RoutineVersionInputModel } from './routineVersionInput';
import { RoutineVersionOutputModel } from './routineVersionOutput';
import { RunProjectModel } from './runProject';
import { RunProjectScheduleModel } from './runProjectSchedule';
import { RunProjectStepModel } from './runProjectStep';
import { RunRoutineModel } from './runRoutine';
import { RunRoutineInputModel } from './runRoutineInput';
import { RunRoutineScheduleModel } from './runRoutineSchedule';
import { RunRoutineStepModel } from './runRoutineStep';
import { SmartContractModel } from './smartContract';
import { SmartContractVersionModel } from './smartContractVersion';
import { StandardModel } from './standard';
import { StandardVersionModel } from './standardVersion';
import { StatsApiModel } from './statsApi';
import { StatsOrganizationModel } from './statsOrganization';
import { StatsProjectModel } from './statsProject';
import { StatsQuizModel } from './statsQuiz';
import { StatsRoutineModel } from './statsRoutine';
import { StatsSiteModel } from './statsSite';
import { StatsSmartContractModel } from './statsSmartContract';
import { StatsStandardModel } from './statsStandard';
import { StatsUserModel } from './statsUser';
import { TagModel } from './tag';
import { TransferModel } from './transfer';
import { ModelLogic } from './types';
import { UserModel } from './user';
import { UserScheduleModel } from './userSchedule';
import { UserScheduleFilterModel } from './userScheduleFilter';
import { ViewModel } from './view';
import { VoteModel } from './vote';
import { WalletModel } from './wallet';

/**
 * Maps model types to their respective business logic implementations.
 */
export const ObjectMap: { [key in GqlModelType]?: ModelLogic<any, any> } = {
    Api: ApiModel,
    ApiKey: ApiKeyModel,
    ApiVersion: ApiVersionModel,
    Award: AwardModel,
    Bookmark: BookmarkModel,
    Comment: CommentModel,
    Email: EmailModel,
    Issue: IssueModel,
    Label: LabelModel,
    Meeting: MeetingModel,
    MeetingInvite: MeetingInviteModel,
    Member: MemberModel,
    MemberInvite: MemberInviteModel,
    Node: NodeModel,
    NodeEnd: NodeEndModel,
    NodeLink: NodeLinkModel,
    NodeLinkWhen: NodeLinkWhenModel,
    NodeLoop: NodeLoopModel,
    NodeLoopWhile: NodeLoopWhileModel,
    NodeRoutineList: NodeRoutineListModel,
    NodeRoutineListItem: NodeRoutineListItemModel,
    Note: NoteModel,
    NoteVersion: NoteVersionModel,
    Notification: NotificationModel,
    NotificationSubscription: NotificationSubscriptionModel,
    Organization: OrganizationModel,
    Payment: PaymentModel,
    Phone: PhoneModel,
    Post: PostModel,
    Premium: PremiumModel,
    Project: ProjectModel,
    ProjectVersion: ProjectVersionModel,
    ProjectVersionDirectory: ProjectVersionDirectoryModel,
    PullRequest: PullRequestModel,
    PushDevice: PushDeviceModel,
    Question: QuestionModel,
    QuestionAnswer: QuestionAnswerModel,
    Quiz: QuizModel,
    QuizAttempt: QuizAttemptModel,
    QuizQuestion: QuizQuestionModel,
    QuizQuestionResponse: QuizQuestionResponseModel,
    Reminder: ReminderModel,
    ReminderItem: ReminderItemModel,
    ReminderList: ReminderListModel,
    Report: ReportModel,
    Resource: ResourceModel,
    ResourceList: ResourceListModel,
    Role: RoleModel,
    Routine: RoutineModel,
    RoutineVersion: RoutineVersionModel,
    RoutineVersionInput: RoutineVersionInputModel,
    RoutineVersionOutput: RoutineVersionOutputModel,
    RunProject: RunProjectModel,
    RunProjectSchedule: RunProjectScheduleModel,
    RunProjectStep: RunProjectStepModel,
    RunRoutine: RunRoutineModel,
    RunRoutineInput: RunRoutineInputModel,
    RunRoutineSchedule: RunRoutineScheduleModel,
    RunRoutineStep: RunRoutineStepModel,
    SmartContract: SmartContractModel,
    SmartContractVersion: SmartContractVersionModel,
    Standard: StandardModel,
    StandardVersion: StandardVersionModel,
    StatsApi: StatsApiModel,
    StatsOrganization: StatsOrganizationModel,
    StatsProject: StatsProjectModel,
    StatsQuiz: StatsQuizModel,
    StatsRoutine: StatsRoutineModel,
    StatsSite: StatsSiteModel,
    StatsSmartContract: StatsSmartContractModel,
    StatsStandard: StatsStandardModel,
    StatsUser: StatsUserModel,
    Tag: TagModel,
    Transfer: TransferModel,
    User: UserModel,
    UserSchedule: UserScheduleModel,
    UserScheduleFilter: UserScheduleFilterModel,
    Vote: VoteModel,
    View: ViewModel,
    Wallet: WalletModel,
}

export * from './api';
export * from './apiKey';
export * from './apiVersion';
export * from './award';
export * from './bookmark';
export * from './comment';
export * from './email';
export * from './routineVersionInput';
export * from './issue';
export * from './label';
export * from './meeting';
export * from './meetingInvite';
export * from './member';
export * from './memberInvite';
export * from './node';
export * from './nodeEnd';
export * from './nodeLink';
export * from './nodeLinkWhen';
export * from './nodeLoop';
export * from './nodeLoopWhile';
export * from './nodeRoutineList';
export * from './nodeRoutineListItem';
export * from './note';
export * from './noteVersion';
export * from './notification';
export * from './notificationSubscription';
export * from './organization';
export * from './routineVersionOutput';
export * from './payment';
export * from './phone';
export * from './post';
export * from './premium';
export * from './project';
export * from './projectVersion';
export * from './projectVersionDirectory';
export * from './pullRequest';
export * from './pushDevice';
export * from './question';
export * from './questionAnswer';
export * from './quiz';
export * from './quizAttempt';
export * from './quizQuestion';
export * from './quizQuestionResponse';
export * from './reminder';
export * from './reminderItem';
export * from './reminderList';
export * from './report';
export * from './resource';
export * from './resourceList';
export * from './role';
export * from './routine';
export * from './routineVersion';
export * from './routineVersionInput';
export * from './routineVersionOutput';
export * from './runProject';
export * from './runProjectSchedule';
export * from './runProjectStep';
export * from './runRoutine';
export * from './runRoutineInput';
export * from './runRoutineSchedule';
export * from './runRoutineStep';
export * from './smartContract';
export * from './smartContractVersion';
export * from './standard';
export * from './standardVersion';
export * from './statsApi';
export * from './statsOrganization';
export * from './statsProject';
export * from './statsQuiz';
export * from './statsRoutine';
export * from './statsSite';
export * from './statsSmartContract';
export * from './statsStandard';
export * from './statsUser';
export * from './runRoutineStep';
export * from './runRoutineInput';
export * from './tag';
export * from './transfer';
export * from './user';
export * from './userSchedule';
export * from './userScheduleFilter';
export * from './view';
export * from './vote';
export * from './wallet';
