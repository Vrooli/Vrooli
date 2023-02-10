import { Question } from "@shared/consts";
import { CreateProps, UpdateProps, ViewProps } from "../types";

export interface QuestionCreateProps extends CreateProps<Question> {}
export interface QuestionUpdateProps extends UpdateProps<Question> {}
export interface QuestionViewProps extends ViewProps<Question> {}