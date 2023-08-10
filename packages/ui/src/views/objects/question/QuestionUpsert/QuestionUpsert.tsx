import { endpointGetQuestion, endpointPostQuestion, endpointPutQuestion, Question, QuestionCreateInput, QuestionUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { QuestionForm, questionInitialValues, transformQuestionValues, validateQuestionValues } from "forms/QuestionForm/QuestionForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { QuestionShape } from "utils/shape/models/question";
import { QuestionUpsertProps } from "../types";

export const QuestionUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex,
}: QuestionUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<Question, QuestionShape>({
        ...endpointGetQuestion,
        objectType: "Question",
        upsertTransform: (existing) => questionInitialValues(session, existing),
    });

    const formRef = useRef<BaseFormRef>();
    const { handleCancel, handleCompleted } = useUpsertActions<Question>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<QuestionCreateInput, Question>(endpointPostQuestion);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<QuestionUpdateInput, Question>(endpointPutQuestion);
    const fetch = (isCreate ? create : update) as MakeLazyRequest<QuestionCreateInput | QuestionUpdateInput, Question>;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateQuestion" : "UpdateQuestion")}
                zIndex={zIndex}
            />
            <Formik
                enableReinitialize={true}
                initialValues={{
                    ...existing,
                    forObject: null,
                }}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<QuestionCreateInput | QuestionUpdateInput, Question>({
                        fetch,
                        inputs: transformQuestionValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateQuestionValues(values, existing)}
            >
                {(formik) => <QuestionForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    );
};
