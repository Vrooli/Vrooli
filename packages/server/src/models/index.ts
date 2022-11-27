import { CommentModel } from './comment';
import { EmailModel } from './email';
import { InputItemModel } from './inputItem';
import { MemberModel } from './member';
import { NodeModel } from './node';
import { NodeEndModel } from './nodeEnd';
import { NodeLinkModel } from './nodeLink';
import { NodeLinkWhenModel } from './nodeLinkWhen';
import { NodeRoutineListModel } from './nodeRoutineList';
import { NodeRoutineListItemModel } from './nodeRoutineListItem';
import { OrganizationModel } from './organization';
import { OutputItemModel } from './outputItem';
import { ProfileModel } from './profile';
import { ProjectModel } from './project';
import { ReportModel } from './report';
import { ResourceModel } from './resource';
import { ResourceListModel } from './resourceList';
import { RoleModel } from './role';
import { RoutineModel } from './routine';
import { RunModel } from './run';
import { RunInputModel } from './runInput';
import { RunStepModel } from './runStep';
import { StandardModel } from './standard';
import { StarModel } from './star';
import { TagModel } from './tag';
import { AniedModelLogic, GraphQLModelType } from './types';
import { UserModel } from './user';
import { ViewModel } from './view';
import { VoteModel } from './vote';
import { WalletModel } from './wallet';

/**
 * Maps model types to various helper functions
 */
export const ObjectMap: { [key in GraphQLModelType]?: AniedModelLogic<any> } = {
    // Api: ApiModel,
    Comment: CommentModel,
    Email: EmailModel,
    InputItem: InputItemModel,
    Member: MemberModel, // TODO create searcher for members
    Node: NodeModel,
    NodeEnd: NodeEndModel,
    NodeLink: NodeLinkModel,
    NodeLinkWhen: NodeLinkWhenModel,
    NodeRoutineList: NodeRoutineListModel,
    NodeRoutineListItem: NodeRoutineListItemModel,
    Organization: OrganizationModel,
    OutputItem: OutputItemModel,
    Profile: ProfileModel,
    Project: ProjectModel,
    Report: ReportModel,
    Resource: ResourceModel,
    ResourceList: ResourceListModel,
    Role: RoleModel,
    Routine: RoutineModel,
    RunRoutine: RunModel,
    RunInput: RunInputModel,
    Standard: StandardModel,
    RunStep: RunStepModel,
    Star: StarModel,
    Tag: TagModel,
    User: UserModel,
    Vote: VoteModel,
    View: ViewModel,
    Wallet: WalletModel,
}

export * from './api';
export * from './comment';
export * from './email';
export * from './inputItem';
export * from './member';
export * from './node';
export * from './nodeEnd';
export * from './nodeLink';
export * from './nodeLinkWhen';
export * from './nodeRoutineList';
export * from './nodeRoutineListItem';
export * from './organization';
export * from './outputItem';
export * from './post';
export * from './profile';
export * from './project';
export * from './report';
export * from './resource';
export * from './resourceList';
export * from './role';
export * from './routine';
export * from './run';
export * from './standard';
export * from './star';
export * from './runStep';
export * from './runInput';
export * from './tag';
export * from './user';
export * from './view';
export * from './vote';
export * from './wallet';
