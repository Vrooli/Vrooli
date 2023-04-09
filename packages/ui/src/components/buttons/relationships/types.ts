import { ObjectType } from "utils/navigation/openObject";

interface RelationshipButtonsBaseProps {
    isEditing: boolean;
    isFormDirty?: boolean;
    objectType: ObjectType;
    zIndex: number;
}

export interface IsCompleteButtonProps extends RelationshipButtonsBaseProps { }
export interface IsPrivateButtonProps extends RelationshipButtonsBaseProps { }
export interface OwnerButtonProps extends RelationshipButtonsBaseProps { }
export interface ParentButtonProps extends RelationshipButtonsBaseProps { }
export interface ProjectButtonProps extends RelationshipButtonsBaseProps { }