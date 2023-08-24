import { endpointGetQuestion, endpointPostQuestion, endpointPutQuestion, Question, QuestionCreateInput, QuestionUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { QuestionForm, questionInitialValues, transformQuestionValues, validateQuestionValues } from "forms/QuestionForm/QuestionForm";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { QuestionShape } from "utils/shape/models/question";
import { QuestionUpsertProps } from "../types";

export const QuestionUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: QuestionUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<Question, QuestionShape>({
        ...endpointGetQuestion,
        objectType: "Question",
        overrideObject,
        transform: (existing) => questionInitialValues(session, existing),
    });

    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<Question, QuestionCreateInput, QuestionUpdateInput>({
        display,
        endpointCreate: endpointPostQuestion,
        endpointUpdate: endpointPutQuestion,
        isCreate,
        onCancel,
        onCompleted,
    });
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    return (
        <MaybeLargeDialog
            display={display}
            id="question-upsert-dialog"
            isOpen={isOpen ?? false}
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
                title={t(isCreate ? "CreateQuestion" : "UpdateQuestion")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={{
                    ...existing,
                    forObject: null,
                } as QuestionShape}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<QuestionCreateInput | QuestionUpdateInput, Question>({
                        fetch,
                        inputs: transformQuestionValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateQuestionValues(values, existing, isCreate)}
            >
                {(formik) => <QuestionForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    onClose={handleClose}
                    ref={formRef}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
