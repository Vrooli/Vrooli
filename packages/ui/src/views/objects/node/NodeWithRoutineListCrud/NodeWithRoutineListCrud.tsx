import { endpointGetApi, nodeTranslationValidation, nodeValidation, noopSubmit } from "@local/shared";
import { Checkbox, FormControlLabel } from "@mui/material";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { EditableTextCollapse } from "components/containers/EditableTextCollapse/EditableTextCollapse";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { DeleteIcon } from "icons";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer } from "styles";
import { getDisplay, getYou } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { shapeNode } from "utils/shape/models/node";
import { validateFormValues } from "utils/validateFormValues";
import { NodeWithRoutineList, NodeWithRoutineListCrudProps, NodeWithRoutineListFormProps, NodeWithRoutineListShape } from "../types";

export const nodeWithRoutineListInitialValues = (existing: NodeWithRoutineListShape): NodeWithRoutineListShape => ({ ...existing });

export const transformNodeWithRoutineListValues = (values: NodeWithRoutineListShape, existing: NodeWithRoutineListShape, isCreate: boolean) =>
    isCreate ? shapeNode.create(values) : shapeNode.update(existing, values);

const NodeWithRoutineListForm = ({
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
    isCreate,
    isEditing,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: NodeWithRoutineListFormProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    // Handle translations
    const {
        language,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["description", "name"],
        validationSchema: nodeTranslationValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" }),
    });

    const [isOrderedField] = useField<boolean>("routineList.isOrdered");
    const [isOptionalField, , isOptionalHelpers] = useField<boolean>("routineList.isOptional");

    const { handleCancel, handleCompleted } = useUpsertActions<NodeWithRoutineListShape>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "Node",
        ...props,
    });
    useSaveToCache({ isCacheOn: false, isCreate, values, objectId: values.id, objectType: "Node" });

    const onSubmit = useCallback(() => {
        handleCompleted(values);
    }, [handleCompleted, values]);

    const isLoading = useMemo(() => isReadLoading || props.isSubmitting, [isReadLoading, props.isSubmitting]);

    return (
        <MaybeLargeDialog
            display={display}
            id="node-with-routine-list-crud-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display="dialog"
                onClose={onClose}
                title={firstString(getDisplay(values).title, t(isCreate ? "CreateNodeRoutineList" : "UpdateNodeRoutineList"))}
                options={!isCreate ? [{
                    Icon: DeleteIcon,
                    label: t("Delete"),
                    onClick: () => { onDeleted?.(existing as NodeWithRoutineList); },
                }] : []}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={500}
            >
                <FormContainer>
                    <EditableTextCollapse
                        component="TranslatedTextInput"
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
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </MaybeLargeDialog>
    );
};

export const NodeWithRoutineListCrud = ({
    isEditing,
    isOpen,
    overrideObject,
    ...props
}: NodeWithRoutineListCrudProps) => {

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<NodeWithRoutineListShape, NodeWithRoutineListShape>({
        ...endpointGetApi, // Won't be used. Need to pass an endpoint to useObjectFromUrl
        isCreate: false,
        objectType: "Node",
        overrideObject: overrideObject as NodeWithRoutineListShape,
        transform: (existing) => nodeWithRoutineListInitialValues(existing as NodeWithRoutineListShape),
    });
    const { canUpdate } = useMemo(() => getYou(existing), [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, false, transformNodeWithRoutineListValues, nodeValidation)}
        >
            {(formik) =>
                <>
                    <NodeWithRoutineListForm
                        disabled={!(canUpdate || isEditing)}
                        existing={existing}
                        handleUpdate={setExisting}
                        isEditing={isEditing}
                        isReadLoading={isReadLoading}
                        isOpen={isOpen}
                        {...props}
                        {...formik}
                    />
                </>
            }
        </Formik>
    );
};
