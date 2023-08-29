import { DUMMY_ID, nodeTranslationValidation, NodeType, nodeValidation, orDefault, Session, uuid } from "@local/shared";
import { Checkbox, FormControlLabel } from "@mui/material";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { EditableTextCollapse } from "components/containers/EditableTextCollapse/EditableTextCollapse";
import { SessionContext } from "contexts/SessionContext";
import { useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { NodeRoutineListFormProps, NodeWithRoutineListShape } from "forms/types";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer } from "styles";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { validateAndGetYupErrors } from "utils/shape/general";
import { shapeNode } from "utils/shape/models/node";

export const nodeRoutineListInitialValues = (
    session: Session | undefined,
    routineVersion: NodeWithRoutineListShape["routineVersion"], // Parent routine version
    existing?: NodeWithRoutineListShape | null | undefined,
): NodeWithRoutineListShape => {
    const id = existing?.id ?? uuid();
    return {
        __typename: "Node" as const,
        id,
        columnIndex: existing?.columnIndex,
        nodeType: NodeType.RoutineList,
        routineList: {
            id: DUMMY_ID,
            __typename: "NodeRoutineList" as const,
            isOptional: false,
            isOrdered: true,
            items: [],
            node: { __typename: "Node" as const, id },
            ...existing?.routineList,
        },
        routineVersion,
        rowIndex: existing?.rowIndex,
        translations: orDefault(existing?.translations, [{
            __typename: "NodeTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            name: "Routine list",
            description: "",
        }]),
    };
};

export const transformNodeRoutineListValues = (values: NodeWithRoutineListShape, existing: NodeWithRoutineListShape, isCreate: boolean) =>
    isCreate ? shapeNode.create(values) : shapeNode.update(existing, values);

export const validateNodeRoutineListValues = async (values: NodeWithRoutineListShape, existing: NodeWithRoutineListShape, isCreate: boolean) => {
    const transformedValues = transformNodeRoutineListValues(values, existing, isCreate);
    const validationSchema = nodeValidation[isCreate ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const NodeRoutineListForm = forwardRef<BaseFormRef | undefined, NodeRoutineListFormProps>(({
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

    const [isOrderedField] = useField<boolean>("routineList.isOrdered");
    const [isOptionalField, , isOptionalHelpers] = useField<boolean>("routineList.isOptional");

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={"dialog"}
                isLoading={isLoading}
                ref={ref}
                maxWidth={500}
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
                    <FormControlLabel
                        disabled={!isEditing}
                        label='Complete in order?'
                        control={
                            <Checkbox
                                size="medium"
                                name='routineList.isOrdered'
                                color='secondary'
                                checked={isOrderedField.value}
                                onChange={isOrderedField.onChange}
                            />
                        }
                    />
                    <FormControlLabel
                        disabled={!isEditing}
                        label='This node is required.'
                        control={
                            <Checkbox
                                size="medium"
                                name='routineList.isOptional'
                                color='secondary'
                                checked={!isOptionalField.value}
                                onChange={(e) => isOptionalHelpers.setValue(!e.target.checked)}
                            />
                        }
                    />
                </FormContainer>
            </BaseForm>
            <BottomActionsButtons
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
