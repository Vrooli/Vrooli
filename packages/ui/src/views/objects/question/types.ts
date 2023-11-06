import { Question } from "@local/shared";
import { FormProps } from "forms/types";
import { QuestionShape } from "utils/shape/models/question";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export type QuestionUpsertProps = UpsertProps<Question>
export type QuestionFormProps = FormProps<Question, QuestionShape>
export type QuestionViewProps = ObjectViewProps<Question>
