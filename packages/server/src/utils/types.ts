// AI_CHECK: TYPE_SAFETY=cudinput-fix | LAST: 2025-07-09 - Fixed CudInputData with proper discriminated union types instead of unknown
import { type DeleteType, type ModelType } from "@vrooli/shared";
import { 
    type ApiKeyCreateInput, type ApiKeyUpdateInput, type ApiKeyExternalCreateInput, type ApiKeyExternalUpdateInput,
    type BookmarkCreateInput, type BookmarkUpdateInput, type BookmarkListCreateInput, type BookmarkListUpdateInput,
    type BotCreateInput, type BotUpdateInput, type ChatCreateInput, type ChatUpdateInput,
    type ChatInviteCreateInput, type ChatInviteUpdateInput, type ChatMessageCreateInput, type ChatMessageUpdateInput,
    type ChatParticipantUpdateInput, type CommentCreateInput, type CommentUpdateInput,
    type EmailCreateInput, type IssueCreateInput, type IssueUpdateInput,
    type MeetingCreateInput, type MeetingUpdateInput, type MeetingInviteCreateInput, type MeetingInviteUpdateInput,
    type MemberUpdateInput, type MemberInviteCreateInput, type MemberInviteUpdateInput,
    type NotificationSettingsUpdateInput, type NotificationSettingsCategoryUpdateInput,
    type NotificationSubscriptionCreateInput, type NotificationSubscriptionUpdateInput,
    type PhoneCreateInput, type ProfileUpdateInput, type ProfileEmailUpdateInput,
    type PullRequestCreateInput, type PullRequestUpdateInput, type PushDeviceCreateInput, type PushDeviceUpdateInput,
    type ReminderCreateInput, type ReminderUpdateInput, type ReminderItemCreateInput, type ReminderItemUpdateInput,
    type ReminderListCreateInput, type ReminderListUpdateInput, type ReportCreateInput, type ReportUpdateInput,
    type ReportResponseCreateInput, type ReportResponseUpdateInput, type ResourceCreateInput, type ResourceUpdateInput,
    type ResourceVersionCreateInput, type ResourceVersionUpdateInput, type ResourceVersionRelationCreateInput, type ResourceVersionRelationUpdateInput,
    type RunCreateInput, type RunUpdateInput, type RunIOCreateInput, type RunIOUpdateInput,
    type RunStepCreateInput, type RunStepUpdateInput, type ScheduleCreateInput, type ScheduleUpdateInput,
    type ScheduleExceptionCreateInput, type ScheduleExceptionUpdateInput, type ScheduleRecurrenceCreateInput, type ScheduleRecurrenceUpdateInput,
    type TagCreateInput, type TagUpdateInput, type TeamCreateInput, type TeamUpdateInput,
    type TransferUpdateInput, type WalletUpdateInput,
} from "@vrooli/shared";
import { type PrismaSelect, type PrismaUpdate } from "../builders/types.js";
import { type InputNode } from "./inputNode.js";

export type QueryAction = "Connect" | "Create" | "Delete" | "Disconnect" | "Read" | "Update";
export type IdsByAction = { [action in QueryAction]?: string[] };
export type IdsByType = { [objectType in ModelType]?: string[] };

// Union types for all Create input types
export type AllCreateInputs = 
    | ApiKeyCreateInput | ApiKeyExternalCreateInput | BookmarkCreateInput | BookmarkListCreateInput
    | BotCreateInput | ChatCreateInput | ChatInviteCreateInput | ChatMessageCreateInput 
    | CommentCreateInput | EmailCreateInput | IssueCreateInput | MeetingCreateInput 
    | MeetingInviteCreateInput | MemberInviteCreateInput | NotificationSubscriptionCreateInput
    | PhoneCreateInput | PullRequestCreateInput | PushDeviceCreateInput | ReminderCreateInput 
    | ReminderItemCreateInput | ReminderListCreateInput | ReportCreateInput | ReportResponseCreateInput
    | ResourceCreateInput | ResourceVersionCreateInput | ResourceVersionRelationCreateInput
    | RunCreateInput | RunIOCreateInput | RunStepCreateInput | ScheduleCreateInput 
    | ScheduleExceptionCreateInput | ScheduleRecurrenceCreateInput | TagCreateInput | TeamCreateInput;

// Union types for all Update input types
export type AllUpdateInputs = 
    | ApiKeyUpdateInput | ApiKeyExternalUpdateInput | BookmarkUpdateInput | BookmarkListUpdateInput
    | BotUpdateInput | ChatUpdateInput | ChatInviteUpdateInput | ChatMessageUpdateInput 
    | ChatParticipantUpdateInput | CommentUpdateInput | IssueUpdateInput | MeetingUpdateInput 
    | MeetingInviteUpdateInput | MemberUpdateInput | MemberInviteUpdateInput | NotificationSettingsUpdateInput
    | NotificationSettingsCategoryUpdateInput | NotificationSubscriptionUpdateInput | ProfileUpdateInput 
    | ProfileEmailUpdateInput | PullRequestUpdateInput | PushDeviceUpdateInput | ReminderUpdateInput 
    | ReminderItemUpdateInput | ReminderListUpdateInput | ReportUpdateInput | ReportResponseUpdateInput
    | ResourceUpdateInput | ResourceVersionUpdateInput | ResourceVersionRelationUpdateInput
    | RunUpdateInput | RunIOUpdateInput | RunStepUpdateInput | ScheduleUpdateInput 
    | ScheduleExceptionUpdateInput | ScheduleRecurrenceUpdateInput | TagUpdateInput | TeamUpdateInput
    | TransferUpdateInput | WalletUpdateInput;
export type InputsById = { [id: string]: { node: InputNode, input: string | AllCreateInputs | AllUpdateInputs } };
export type InputsByType = { [objectType in ModelType]?: {
    Connect: { node: InputNode, input: string | bigint; }[];
    Create: { node: InputNode, input: PrismaUpdate }[];
    Delete: { node: InputNode, input: string | bigint; }[];
    Disconnect: { node: InputNode, input: string | bigint; }[];
    Read: { node: InputNode, input: PrismaSelect }[];
    Update: { node: InputNode, input: PrismaUpdate }[];
} };
export type IdsByPlaceholder = { [placeholder: string]: string | bigint | null };
export type IdsCreateToConnect = { [id: string]: string };
export type CudInputData = {
    action: "Create";
    input: AllCreateInputs;
    objectType: ModelType | `${ModelType}`;
} | {
    action: "Update";
    input: AllUpdateInputs;
    objectType: ModelType | `${ModelType}`;
} | {
    action: "Delete";
    input: string | bigint;
    objectType: DeleteType | `${DeleteType}`;
};
export type ResultsById = { [id: string]: unknown };
