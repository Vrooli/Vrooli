import { FindByIdInput, Question, QuestionCreateInput, QuestionUpdateInput } from "@local/shared";
import { mutationWrapper } from "api";
import { questionCreate } from "api/generated/endpoints/question_create";
import { questionFindOne } from "api/generated/endpoints/question_findOne";
import { questionUpdate } from "api/generated/endpoints/question_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { QuestionForm, questionInitialValues, transformQuestionValues, validateQuestionValues } from "forms/QuestionForm/QuestionForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { QuestionUpsertProps } from "../types";

export const QuestionUpsert = ({
    display = 'page',
    isCreate,
    onCancel,
    onCompleted,
    zIndex = 200,
}: QuestionUpsertProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<Question, FindByIdInput>(questionFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => questionInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<Question>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation<Question, QuestionCreateInput>(questionCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation<Question, QuestionUpdateInput>(questionUpdate);
    const mutation = isCreate ? create : update;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: isCreate ? 'CreateQuestion' : 'UpdateQuestion',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: 'CouldNotReadObject', severity: 'Error' });
                        return;
                    }
                    mutationWrapper<Question, QuestionCreateInput | QuestionUpdateInput>({
                        mutation,
                        input: transformQuestionValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
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
    )
}