import { Question, QuestionCreateInput } from "@shared/consts";
import { questionValidation } from '@shared/validation';
import { mutationWrapper } from "api";
import { questionCreate } from "api/generated/endpoints/question_create";
import { useCustomMutation } from "api/hooks";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { QuestionForm } from "forms/QuestionForm/QuestionForm";
import { useContext, useMemo, useRef } from "react";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { SessionContext } from "utils/SessionContext";
import { shapeQuestion } from "utils/shape/models/question";
import { questionInitialValues } from "..";
import { QuestionCreateProps } from "../types";

export const QuestionCreate = ({
    display = 'page',
    onCancel,
    onCreated,
    zIndex = 200,
}: QuestionCreateProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => questionInitialValues(session), [session]);
    const { handleCancel, handleCreated } = useCreateActions<Question>(display, onCancel, onCreated);
    const [mutation, { loading: isLoading }] = useCustomMutation<Question, QuestionCreateInput>(questionCreate);

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: 'CreateQuestion',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    mutationWrapper<Question, QuestionCreateInput>({
                        mutation,
                        input: shapeQuestion.create(values),
                        onSuccess: (data) => { handleCreated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validationSchema={questionValidation.create({})}
            >
                {(formik) => <QuestionForm
                    display={display}
                    isCreate={true}
                    isLoading={isLoading}
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