import { ObjectType } from "utils/navigation/openObject";

interface RelationshipButtonsBaseProps {
    isEditing: boolean;
    isFormDirty?: boolean;
    objectType: ObjectType;
}

export type FocusModeButtonProps = RelationshipButtonsBaseProps
export type IsCompleteButtonProps = RelationshipButtonsBaseProps
export type IsPrivateButtonProps = RelationshipButtonsBaseProps
export type MeetingButtonProps = RelationshipButtonsBaseProps
export type MembersButtonProps = RelationshipButtonsBaseProps
export type OwnerButtonProps = RelationshipButtonsBaseProps
export type ParentButtonProps = RelationshipButtonsBaseProps
export type ParticipantsButtonProps = RelationshipButtonsBaseProps
export type ProjectButtonProps = RelationshipButtonsBaseProps
export type QuestionForButtonProps = RelationshipButtonsBaseProps
export type RunProjectButtonProps = RelationshipButtonsBaseProps
export type RunRoutineButtonProps = RelationshipButtonsBaseProps
export type TeamButtonProps = RelationshipButtonsBaseProps
export type UserButtonProps = RelationshipButtonsBaseProps
