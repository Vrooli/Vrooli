import { DUMMY_ID, nodeTranslationValidation, NodeType, nodeValidation, orDefault, RoutineVersion, Session, uuid } from "@local/shared";
import { Checkbox, FormControlLabel, Tooltip } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { EditableTextCollapse } from "components/containers/EditableTextCollapse/EditableTextCollapse";
import { SessionContext } from "contexts/SessionContext";
import { useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { NodeEndFormProps, NodeWithEndShape } from "forms/types";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer } from "styles";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { validateAndGetYupErrors } from "utils/shape/general";
import { shapeNode } from "utils/shape/models/node";

export const nodeEndInitialValues = (
    session: Session | undefined,
    routineVersion: RoutineVersion,
    existing?: NodeWithEndShape | null | undefined,
): NodeWithEndShape => {
    const id = uuid();
    return {
        __typename: "Node" as const,
        id,
        end: {
            id: DUMMY_ID,
            __typename: "NodeEnd" as const,
            node: { __typename: "Node" as const, id },
            suggestedNextRoutineVersions: [],
            wasSuccessful: true,
        },
        nodeType: NodeType.End,
        routineVersion,
        ...existing,
        translations: orDefault(existing?.translations, [{
            __typename: "NodeTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            name: "End",
            description: "",
        }]),
    };
};

export const transformNodeEndValues = (values: NodeWithEndShape, existing: NodeWithEndShape, isCreate: boolean) =>
    isCreate ? shapeNode.create(values) : shapeNode.update(existing, values);

export const validateNodeEndValues = async (values: NodeWithEndShape, existing: NodeWithEndShape, isCreate: boolean) => {
    const transformedValues = transformNodeEndValues(values, existing, isCreate);
    const validationSchema = nodeValidation[isCreate ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const NodeEndForm = forwardRef<BaseFormRef | undefined, NodeEndFormProps>(({
    display,
    dirty,
    isCreate,
    isEditing,
    isLoading,
    isOpen,
    onCancel,
    values,
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
        fields: ["description", "name"],
        validationSchema: nodeTranslationValidation[isCreate ? "create" : "update"]({}),
    });

    const [wasSuccessfulField] = useField<boolean>("end.wasSuccessful");

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                maxWidth={500}
                ref={ref}
            >
                <FormContainer>
                    <EditableTextCollapse
                        component='TranslatedTextField'
                        isEditing={isEditing}
                        name="name"
                        props={{
                            language,
                            fullWidth: true,
                            multiline: true,
                        }}
                        title={t("Label")}
                    />
                    <EditableTextCollapse
                        component='TranslatedMarkdown'
                        isEditing={isEditing}
                        name="description"
                        props={{
                            language,
                            maxChars: 2048,
                            minRows: 4,
                            maxRows: 8,
                        }}
                        title={t("Description")}
                    />
                    <Tooltip placement={"top"} title={t("NodeWasSuccessfulHelp")}>
                        <FormControlLabel
                            disabled={!isEditing}
                            label='Success?'
                            control={
                                <Checkbox
                                    id={"end-node-was-successful"}
                                    size="medium"
                                    name='end.wasSuccessful'
                                    color='secondary'
                                    checked={wasSuccessfulField.value}
                                    onChange={wasSuccessfulField.onChange}
                                />
                            }
                        />
                    </Tooltip>
                </FormContainer>
            </BaseForm>
            <GridSubmitButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
});
