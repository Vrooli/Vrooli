import { ApiModel } from './api';
import { ApiKeyModel } from './apiKey';
import { ApiVersionModel } from './apiVersion';
import { CommentModel } from './comment';
import { EmailModel } from './email';
import { InputItemModel } from './inputItem';
import { IssueModel } from './issue';
import { LabelModel } from './label';
import { MemberModel } from './member';
import { NodeModel } from './node';
import { NodeEndModel } from './nodeEnd';
import { NodeLinkModel } from './nodeLink';
import { NodeLinkWhenModel } from './nodeLinkWhen';
import { NodeRoutineListModel } from './nodeRoutineList';
import { NodeRoutineListItemModel } from './nodeRoutineListItem';
import { OrganizationModel } from './organization';
import { OutputItemModel } from './outputItem';
import { PhoneModel } from './phone';
import { ProjectModel } from './project';
import { ProjectVersionModel } from './projectVersion';
import { QuestionModel } from './question';
import { QuestionAnswerModel } from './questionAnswer';
import { ReminderModel } from './reminder';
import { ReminderListModel } from './reminderList';
import { ReportModel } from './report';
import { ResourceModel } from './resource';
import { ResourceListModel } from './resourceList';
import { RoleModel } from './role';
import { RoutineModel } from './routine';
import { RoutineVersionModel } from './routineVersion';
import { RunModel } from './run';
import { RunInputModel } from './runInput';
import { RunStepModel } from './runStep';
import { SmartContractModel } from './smartContract';
import { SmartContractVersionModel } from './smartContractVersion';
import { StandardModel } from './standard';
import { StandardVersionModel } from './standardVersion';
import { StarModel } from './star';
import { TagModel } from './tag';
import { AniedModelLogic, GraphQLModelType } from './types';
import { UserModel } from './user';
import { ViewModel } from './view';
import { VoteModel } from './vote';
import { WalletModel } from './wallet';

/**
 * Maps model types to their respective business logic implementations.
 */
export const ObjectMap: { [key in GraphQLModelType]?: AniedModelLogic<any> } = {
    Api: ApiModel,
    ApiKey: ApiKeyModel,
    ApiVersion: ApiVersionModel,
    Comment: CommentModel,
    Email: EmailModel,
    InputItem: InputItemModel,
    Issue: IssueModel,
    Label: LabelModel,
    Member: MemberModel,
    Node: NodeModel,
    NodeEnd: NodeEndModel,
    NodeLink: NodeLinkModel,
    NodeLinkWhen: NodeLinkWhenModel,
    NodeRoutineList: NodeRoutineListModel,
    NodeRoutineListItem: NodeRoutineListItemModel,
    Organization: OrganizationModel,
    OutputItem: OutputItemModel,
    Phone: PhoneModel,
    Project: ProjectModel,
    ProjectVersion: ProjectVersionModel,
    Question: QuestionModel,
    QuestionAnswer: QuestionAnswerModel,
    Reminder: ReminderModel,
    ReminderList: ReminderListModel,
    Report: ReportModel,
    Resource: ResourceModel,
    ResourceList: ResourceListModel,
    Role: RoleModel,
    Routine: RoutineModel,
    RoutineVersion: RoutineVersionModel,
    RunRoutine: RunModel,
    RunInput: RunInputModel,
    RunStep: RunStepModel,
    SmartContract: SmartContractModel,
    SmartContractVersion: SmartContractVersionModel,
    Standard: StandardModel,
    StandardVersion: StandardVersionModel,
    Star: StarModel,
    Tag: TagModel,
    User: UserModel,
    Vote: VoteModel,
    View: ViewModel,
    Wallet: WalletModel,
}

export * from './api';
export * from './apiKey';
export * from './apiVersion';
export * from './comment';
export * from './email';
export * from './inputItem';
export * from './issue';
export * from './label';
export * from './member';
export * from './node';
export * from './nodeEnd';
export * from './nodeLink';
export * from './nodeLinkWhen';
export * from './nodeRoutineList';
export * from './nodeRoutineListItem';
export * from './organization';
export * from './outputItem';
export * from './phone';
export * from './post';
export * from './profile';
export * from './project';
export * from './projectVersion';
export * from './question';
export * from './questionAnswer';
export * from './reminder';
export * from './reminderList';
export * from './report';
export * from './resource';
export * from './resourceList';
export * from './role';
export * from './routine';
export * from './routineVersion';
export * from './run';
export * from './runInput';
export * from './runStep';
export * from './smartContract';
export * from './smartContractVersion';
export * from './standard';
export * from './standardVersion';
export * from './star';
export * from './runStep';
export * from './runInput';
export * from './tag';
export * from './user';
export * from './view';
export * from './vote';
export * from './wallet';
