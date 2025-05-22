/* c8 ignore start */
import { type ObjectType } from "../../../utils/navigation/openObject.js";

interface RelationshipButtonsBaseProps {
    isEditing: boolean;
    isFormDirty?: boolean;
    objectType: ObjectType;
}

export type IsCompleteButtonProps = RelationshipButtonsBaseProps
export type IsPrivateButtonProps = RelationshipButtonsBaseProps
export type MembersButtonProps = RelationshipButtonsBaseProps
export type OwnerButtonProps = RelationshipButtonsBaseProps
export type ParticipantsButtonProps = RelationshipButtonsBaseProps
export type TeamButtonProps = RelationshipButtonsBaseProps
export type UserButtonProps = RelationshipButtonsBaseProps
