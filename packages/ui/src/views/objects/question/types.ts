import { Question, QuestionShape } from "@local/shared";
import { FormProps } from "forms/types";
import { ObjectViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type QuestionUpsertPropsPage = CrudPropsPage;
type QuestionUpsertPropsDialog = CrudPropsDialog<Question>;
export type QuestionUpsertProps = QuestionUpsertPropsPage | QuestionUpsertPropsDialog;
export type QuestionFormProps = FormProps<Question, QuestionShape>
export type QuestionViewProps = ObjectViewProps<Question>
