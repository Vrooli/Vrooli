import { Question, QuestionShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps, ObjectViewProps } from "../../../types.js";

type QuestionUpsertPropsPage = CrudPropsPage;
type QuestionUpsertPropsDialog = CrudPropsDialog<Question>;
export type QuestionUpsertProps = QuestionUpsertPropsPage | QuestionUpsertPropsDialog;
export type QuestionFormProps = FormProps<Question, QuestionShape>
export type QuestionViewProps = ObjectViewProps<Question>
