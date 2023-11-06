import { Comment, CommentFor } from "@local/shared";
import { FormProps } from "forms/types";
import { CommentShape } from "utils/shape/models/comment";
import { UpsertProps } from "../types";

export type CommentUpsertProps = Omit<UpsertProps<Comment>, "isCreate" | "onCancel" | "onClose" | "onCompleted"> & Required<Pick<UpsertProps<Comment>, "onCancel" | "onClose" | "onCompleted">> & {
    language: string;
    objectId: string;
    objectType: CommentFor;
    parent: Comment | null;
}
export type CommentFormProps = Omit<FormProps<Comment, CommentShape>, "onCancel" | "onClose" | "onCompleted"> & Pick<CommentUpsertProps, "language" | "objectId" | "objectType" | "onCancel" | "onClose" | "onCompleted" | "parent">;
