import { Question, QuestionCreateInput } from "@shared/consts";
import { uuid } from '@shared/uuid';
import { questionValidation } from '@shared/validation';
import { mutationWrapper } from "api";
import { questionCreate } from "api/generated/endpoints/question_create";
import { useCustomMutation } from "api/hooks";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { QuestionForm } from "forms/QuestionForm/QuestionForm";
import { useContext, useRef } from "react";
import { getUserLanguages } from "utils/display/translationTools";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { SessionContext } from "utils/SessionContext";
import { shapeQuestion } from "utils/shape/models/question";
import { QuestionCreateProps } from "../types";

export const QuestionCreate = ({
    display = 'page',
    zIndex = 200,
}: QuestionCreateProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const { onCancel, onCreated } = useCreateActions<Question>();
    const [mutation, { loading: isLoading }] = useCustomMutation<Question, QuestionCreateInput>(questionCreate);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'CreateQuestion',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={{
                    __typename: 'Question' as const,
                    id: uuid(),
                    translations: [{
                        id: uuid(),
                        language: getUserLanguages(session)[0],
                        description: '',
                        name: '',
                    }]
                }}
                onSubmit={(values, helpers) => {
                    mutationWrapper<Question, QuestionCreateInput>({
                        mutation,
                        input: shapeQuestion.create(values),
                        onSuccess: (data) => { onCreated(data) },
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
                    onCancel={onCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    )
}