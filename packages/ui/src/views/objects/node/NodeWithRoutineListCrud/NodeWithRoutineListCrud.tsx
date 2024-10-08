import { endpointGetApi, nodeTranslationValidation, nodeValidation, noopSubmit, shapeNode } from "@local/shared";
import { Checkbox, FormControlLabel } from "@mui/material";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { EditableTextCollapse } from "components/containers/EditableTextCollapse/EditableTextCollapse";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts";
import { Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useManagedObject } from "hooks/useManagedObject";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { DeleteIcon } from "icons";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer } from "styles";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { validateFormValues } from "utils/validateFormValues";
import { NodeWithRoutineList, NodeWithRoutineListCrudProps, NodeWithRoutineListFormProps, NodeWithRoutineListShape } from "../types";

export function nodeWithRoutineListInitialValues(existing: NodeWithRoutineListShape): NodeWithRoutineListShape {
    return { ...existing };
}

export function transformNodeWithRoutineListValues(values: NodeWithRoutineListShape, existing: NodeWithRoutineListShape, isCreate: boolean) {
    return isCreate ? shapeNode.create(values) : shapeNode.update(existing, values);
}

function NodeWithRoutineListForm({
    disabled,
    display,
    existing,
    isCreate,
    isEditing,
    isOpen,
    isReadLoading,
    onCompleted,
    onClose,
    onDeleted,
    values,
    ...props
}: NodeWithRoutineListFormProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    // Handle translations
    const {
        language,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        validationSchema: nodeTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    const [isOrderedField] = useField<boolean>("routineList.isOrdered");
    const [isOptionalField, , isOptionalHelpers] = useField<boolean>("routineList.isOptional");

    const toggleIsOptional = useCallback(function toggleIsOptionalCallback() {
        isOptionalHelpers.setValue(!isOptionalField.value);
    }, [isOptionalField, isOptionalHelpers]);

    const { handleCancel, handleCompleted } = useUpsertActions<NodeWithRoutineListShape>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "Node",
        onAction: onClose,
        onCompleted: onCompleted as (data: NodeWithRoutineListShape) => unknown,
        onDeleted: onDeleted as (data: NodeWithRoutineListShape) => unknown,
        suppressSnack: true,
        ...props,
    });
    useSaveToCache({ isCacheOn: false, isCreate, values, objectId: values.id, objectType: "Node" });

    const onSubmit = useCallback(() => {
        handleCompleted(values);
    }, [handleCompleted, values]);

    const isLoading = useMemo(() => isReadLoading || props.isSubmitting, [isReadLoading, props.isSubmitting]);

    const topBarOptions = useMemo(function topBarOptionsMemo() {
        return isCreate ? [] : [{
            Icon: DeleteIcon,
            label: t("Delete"),
            onClick: () => { onDeleted?.(existing as NodeWithRoutineList); },
        }];
    }, [existing, isCreate, onDeleted, t]);

    const nameInputProps = useMemo(function nameInputPropsMemo() {
        return {
            language,
            fullWidth: true,
            multiline: false,
        } as const;
    }, [language]);

    const descriptionInputProps = useMemo(function descriptionInputPropsMemo() {
        return {
            language,
            maxChars: 2048,
            minRows: 4,
            maxRows: 8,
        } as const;
    }, [language]);

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
                options={topBarOptions}
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
                        props={nameInputProps}
                        title={t("Label")}
                    />
                    <EditableTextCollapse
                        component='TranslatedMarkdown'
                        isEditing={isEditing}
                        name="description"
                        props={descriptionInputProps}
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
                                onChange={toggleIsOptional}
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
}

export function NodeWithRoutineListCrud({
    isEditing,
    isOpen,
    overrideObject,
    ...props
}: NodeWithRoutineListCrudProps) {

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<NodeWithRoutineListShape, NodeWithRoutineListShape>({
        ...endpointGetApi, // Won't be used. Need to pass an endpoint to useManagedObject
        isCreate: false,
        objectType: "Node",
        overrideObject: overrideObject as NodeWithRoutineListShape,
        transform: (existing) => nodeWithRoutineListInitialValues(existing as NodeWithRoutineListShape),
    });

    async function validateValues(values: NodeWithRoutineListShape) {
        return await validateFormValues(values, existing, false, transformNodeWithRoutineListValues, nodeValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) =>
                <>
                    <NodeWithRoutineListForm
                        disabled={!(permissions.canUpdate || isEditing)}
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
}
