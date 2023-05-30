import { DUMMY_ID, nodeTranslationValidation, NodeType, nodeValidation, orDefault, RoutineVersion, Session, uuid } from "@local/shared";
import { Checkbox, FormControlLabel, Stack, Tooltip } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { EditableTextCollapse } from "components/containers/EditableTextCollapse/EditableTextCollapse";
import { useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { NodeRoutineListFormProps, NodeWithRoutineListShape } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { shapeNode } from "utils/shape/models/node";

export const nodeRoutineListInitialValues = (
    session: Session | undefined,
    routineVersion: RoutineVersion,
    existing?: NodeWithRoutineListShape | null | undefined,
): NodeWithRoutineListShape => {
    const id = uuid();
    return {
        __typename: "Node" as const,
        id,
        nodeType: NodeType.RoutineList,
        routineList: {
            id: DUMMY_ID,
            __typename: "NodeRoutineList" as const,
            isOptional: false,
            isOrdered: true,
            items: [],
            node: { __typename: "Node" as const, id },
        },
        routineVersion,
        ...existing,
        translations: orDefault(existing?.translations, [{
            __typename: "NodeTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            name: "Routine list",
            description: "",
        }]),
    };
};

export const transformNodeRoutineListValues = (values: NodeWithRoutineListShape, existing?: NodeWithRoutineListShape) => {
    return existing === undefined
        ? shapeNode.create(values)
        : shapeNode.update(existing, values);
};

export const validateNodeRoutineListValues = async (values: NodeWithRoutineListShape, existing?: NodeWithRoutineListShape) => {
    const transformedValues = transformNodeRoutineListValues(values, existing);
    const validationSchema = nodeValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const NodeRoutineListForm = forwardRef<any, NodeRoutineListFormProps>(({
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
        fields: ["description", "name"],
        validationSchema: nodeTranslationValidation[isCreate ? "create" : "update"]({}),
    });

    const [isOrderedField] = useField<boolean>("routineList.isOrdered");
    const [isOptionalField] = useField<boolean>("routineList.isOptional");

    return (
        <>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: "block",
                    minWidth: "400px",
                    maxWidth: "700px",
                    marginBottom: "64px",
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
                        title={t("Label")}
                    />
                    <EditableTextCollapse
                        component='TranslatedMarkdown'
                        isEditing={isEditing}
                        name="description"
                        props={{
                            language,
                            zIndex,
                        }}
                        title={t("Description")}
                    />
                    <Tooltip placement={"top"} title={t("MustCompleteRoutinesInOrder")}>
                        <FormControlLabel
                            disabled={!isEditing}
                            label='Complete in order?'
                            control={
                                <Checkbox
                                    id={"routine-list-node-is-ordered"}
                                    size="medium"
                                    name='routineList.isOrdered'
                                    color='secondary'
                                    checked={isOrderedField.value}
                                    onChange={isOrderedField.onChange}
                                />
                            }
                        />
                    </Tooltip>
                    <Tooltip placement={"top"} title={t("RoutineCanSkip")}>
                        <FormControlLabel
                            disabled={!isEditing}
                            label='This node is required.'
                            control={
                                <Checkbox
                                    id={"routine-list-node-is-optional"}
                                    size="medium"
                                    name='routineList.isOptional'
                                    color='secondary'
                                    checked={!isOptionalField.value}
                                    onChange={isOptionalField.onChange}
                                />
                            }
                        />
                    </Tooltip>
                </Stack>
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
