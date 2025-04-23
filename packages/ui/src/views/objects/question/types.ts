import { Question, QuestionShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, CrudPropsPartial, FormProps, ObjectViewProps } from "../../../types.js";

type QuestionUpsertPropsPage = CrudPropsPage;
type QuestionUpsertPropsDialog = CrudPropsDialog<Question>;
type QuestionUpsertPropsPartial = CrudPropsPartial<Question>;
export type QuestionUpsertProps = QuestionUpsertPropsPage | QuestionUpsertPropsDialog | QuestionUpsertPropsPartial;
export type QuestionFormProps = FormProps<Question, QuestionShape>
export type QuestionViewProps = ObjectViewProps<Question>
