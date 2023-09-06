import { Question } from "@local/shared";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export type QuestionUpsertProps = UpsertProps<Question>
export type QuestionViewProps = ObjectViewProps<Question>
