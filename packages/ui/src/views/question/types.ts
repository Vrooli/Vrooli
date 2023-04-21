import { Question } from "@shared/consts";
import { UpsertProps, ViewProps } from "../types";

export interface QuestionUpsertProps extends UpsertProps<Question> { }
export interface QuestionViewProps extends ViewProps<Question> { }