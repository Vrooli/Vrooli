import { FindByIdInput, Question, QuestionUpdateInput } from "@shared/consts";
import { questionValidation } from '@shared/validation';
import { mutationWrapper } from "api";
import { questionFindOne } from "api/generated/endpoints/question_findOne";
import { questionUpdate } from "api/generated/endpoints/question_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { QuestionForm } from "forms/QuestionForm/QuestionForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { shapeQuestion } from "utils/shape/models/question";
import { questionInitialValues } from "..";
import { QuestionUpdateProps } from "../types";

export const QuestionUpdate = ({
    display = 'page',
    onCancel,
    onUpdated,
    zIndex = 200,
}: QuestionUpdateProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<Question, FindByIdInput>(questionFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => questionInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleUpdated } = useUpdateActions<Question>(display, onCancel, onUpdated);
    const [mutation, { loading: isUpdateLoading }] = useCustomMutation<Question, QuestionUpdateInput>(questionUpdate);

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: 'UpdateQuestion',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!existing) {
                        PubSub.get().publishSnack({ messageKey: 'CouldNotReadObject', severity: 'Error' });
                        return;
                    }
                    mutationWrapper<Question, QuestionUpdateInput>({
                        mutation,
                        input: shapeQuestion.update(existing, values),
                        onSuccess: (data) => { handleUpdated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validationSchema={questionValidation.update({})}
            >
                {(formik) => <QuestionForm
                    display={display}
                    isCreate={false}
                    isLoading={isReadLoading || isUpdateLoading}
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