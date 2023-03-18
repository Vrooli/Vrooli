import { Grid } from "@mui/material";
import { Question, QuestionCreateInput, ResourceList } from "@shared/consts";
import { parseSearchParams } from "@shared/route";
import { uuid } from '@shared/uuid';
import { questionTranslationValidation, questionValidation } from '@shared/validation';
import { questionCreate } from "api/generated/endpoints/question_create";
import { useCustomMutation } from "api/hooks";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { RelationshipButtons } from "components/inputs/RelationshipButtons/RelationshipButtons";
import { RelationshipsObject } from "components/inputs/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useFormik } from 'formik';
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { checkIfLoggedIn } from "utils/authentication/session";
import { defaultRelationships } from "utils/defaults/relationships";
import { defaultResourceList } from "utils/defaults/resourceList";
import { getUserLanguages } from "utils/display/translationTools";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { TagShape } from "utils/shape/models/tag";
import { QuestionCreateProps } from "../types";

export const QuestionCreate = ({
    display = 'page',
    zIndex = 200,
}: QuestionCreateProps) => {
    const session = useContext(SessionContext);

    const { onCancel, onCreated } = useCreateActions<Question>();

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>(defaultRelationships(true, session));
    const onRelationshipsChange = useCallback((change: Partial<RelationshipsObject>) => setRelationships({ ...relationships, ...change }), [relationships]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>(defaultResourceList);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => setResourceList(updatedList), [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        const params = parseSearchParams();
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, []);

    // Handle create
    const [mutation] = useCustomMutation<Question, QuestionCreateInput>(questionCreate);
    const formik = useFormik({
        initialValues: {
            id: uuid(),
            translationsCreate: [{
                id: uuid(),
                language: getUserLanguages(session)[0],
                description: '',
                name: '',
            }]
        },
        validationSchema: questionValidation.create({}),
        onSubmit: (values) => {
            // mutationWrapper<Question, QuestionCreateInput>({
            //     mutation,
            //     input: shapeQuestion.create({
            //         id: values.id,

            //     }),
            //     onSuccess: (data) => { onCreated(data) },
            //     onError: () => { formik.setSubmitting(false) },
            // })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        onTranslationBlur,
        onTranslationChange,
        setLanguage,
        translations,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ['description', 'name'],
        formik,
        formikField: 'translationsCreate',
        validationSchema: questionTranslationValidation.create({}),
    });

    const isLoggedIn = useMemo(() => checkIfLoggedIn(session), [session]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'CreateQuestion',
                }}
            />
            <BaseForm onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                    <Grid item xs={12} mb={4}>
                        <RelationshipButtons
                            isEditing={true}
                            objectType={'Question'}
                            onRelationshipsChange={onRelationshipsChange}
                            relationships={relationships}
                            zIndex={zIndex}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            translations={formik.values.translationsCreate}
                            zIndex={zIndex}
                        />
                    </Grid>
                    {/* TODO */}
                    <GridSubmitButtons
                        disabledSubmit={!isLoggedIn}
                        display={display}
                        errors={translations.errorsWithTranslations}
                        isCreate={true}
                        loading={formik.isSubmitting}
                        onCancel={onCancel}
                        onSetSubmitting={formik.setSubmitting}
                        onSubmit={formik.handleSubmit}
                    />
                </Grid>
            </BaseForm>
        </>
    )
}