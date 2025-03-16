import { Comment, CommentFor, CommentShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPartial, FormProps } from "../../../types.js";

type CommentUpsertPropsBase = {
    language: string;
    objectId: string;
    objectType: CommentFor;
    parent: Comment | null;
};
type CommentUpsertPropsDialog = CommentUpsertPropsBase & CrudPropsDialog<Comment>;
type CommentUpsertPropsPartial = CommentUpsertPropsBase & CrudPropsPartial<Comment>;
export type CommentUpsertProps = CommentUpsertPropsDialog | CommentUpsertPropsPartial;
export type CommentFormProps = Omit<FormProps<Comment, CommentShape>, "onCancel" | "onClose" | "onCompleted"> & Pick<CommentUpsertProps, "language" | "objectId" | "objectType" | "onCancel" | "onClose" | "onCompleted" | "parent">;
