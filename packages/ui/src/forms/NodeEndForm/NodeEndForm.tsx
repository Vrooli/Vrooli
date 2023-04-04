import { Checkbox, FormControlLabel, Stack, Tooltip } from "@mui/material";
import { NodeType, RoutineVersion, Session } from "@shared/consts";
import { DUMMY_ID, uuid } from "@shared/uuid";
import { nodeTranslationValidation, nodeValidation } from "@shared/validation";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { EditableTextCollapse } from "components/containers/EditableTextCollapse/EditableTextCollapse";
import { useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { NodeEndFormProps, NodeWithEndShape } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { shapeNode } from "utils/shape/models/node";

export const nodeEndInitialValues = (
    session: Session | undefined,
    routineVersion: RoutineVersion,
    existing?: NodeWithEndShape | null | undefined
): NodeWithEndShape => {
    const id = uuid();
    return {
        __typename: 'Node' as const,
        id,
        end: {
            id: DUMMY_ID,
            __typename: 'NodeEnd' as const,
            node: { __typename: 'Node' as const, id },
            suggestedNextRoutineVersions: [],
            wasSuccessful: true,
        },
        nodeType: NodeType.End,
        routineVersion,
        translations: [{
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            name: 'End',
            description: '',
        }],
        ...existing,
    }
};

export const transformNodeEndValues = (values: NodeWithEndShape, existing?: NodeWithEndShape) => {
    return existing === undefined
        ? shapeNode.create(values)
        : shapeNode.update(existing, values)
}

export const validateNodeEndValues = async (values: NodeWithEndShape, existing?: NodeWithEndShape) => {
    const transformedValues = transformNodeEndValues(values, existing);
    const validationSchema = existing === undefined
        ? nodeValidation.create({})
        : nodeValidation.update({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
}

export const NodeEndForm = forwardRef<any, NodeEndFormProps>(({
    display,
    dirty,
    isCreate,
    isEditing,
    isLoading,
    isOpen,
    onCancel,
    values,
    zIndex,
    ...props
}, ref) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    // Handle translations
    const {
        language,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ['description', 'name'],
        validationSchema: isCreate ? nodeTranslationValidation.create({}) : nodeTranslationValidation.update({}),
    });

    const [wasSuccessfulField] = useField<boolean>('end.wasSuccessful');

    return (
        <>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: 'block',
                    minWidth: '400px',
                    maxWidth: '700px',
                    marginBottom: '64px',
                }}
            >
                <Stack direction="column" spacing={4} sx={{
                    margin: 2,
                    marginBottom: 4,
                }}>
                    <EditableTextCollapse
                        component='TranslatedTextField'
                        isEditing={isEditing}
                        name="name"
                        props={{
                            language,
                            fullWidth: true,
                            multiline: true,
                        }}
                        title={t('Label')}
                    />
                    <EditableTextCollapse
                        component='TranslatedMarkdown'
                        isEditing={isEditing}
                        name="description"
                        props={{
                            language,
                        }}
                        title={t('Description')}
                    />
                    <Tooltip placement={'top'} title={t('NodeWasSuccessfulHelp')}>
                        <FormControlLabel
                            disabled={!isEditing}
                            label='Success?'
                            control={
                                <Checkbox
                                    id={`end-node-was-successful`}
                                    size="medium"
                                    name='end.wasSuccessful'
                                    color='secondary'
                                    checked={wasSuccessfulField.value}
                                    onChange={wasSuccessfulField.onChange}
                                />
                            }
                        />
                    </Tooltip>
                </Stack>
            </BaseForm>
            <GridSubmitButtons
                display={display}
                errors={{
                    ...props.errors,
                    ...translationErrors,
                }}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    )
})