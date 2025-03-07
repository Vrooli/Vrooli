import { endpointsApi, nodeTranslationValidation, nodeValidation, noopSubmit, shapeNode } from "@local/shared";
import { Checkbox, FormControlLabel, Tooltip } from "@mui/material";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons.js";
import { EditableTextCollapse } from "components/containers/EditableTextCollapse/EditableTextCollapse.js";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog.js";
import { TopBar } from "components/navigation/TopBar/TopBar.js";
import { Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "hooks/forms.js";
import { useManagedObject } from "hooks/useManagedObject.js";
import { useTranslatedFields } from "hooks/useTranslatedFields.js";
import { DeleteIcon } from "icons/common.js";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools.js";
import { firstString } from "utils/display/stringTools.js";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools.js";
import { validateFormValues } from "utils/validateFormValues.js";
import { SessionContext } from "../../../../contexts.js";
import { FormContainer } from "../../../../styles.js";
import { NodeWithEnd, NodeWithEndCrudProps, NodeWithEndFormProps, NodeWithEndShape } from "../types.js";

export function nodeWithEndInitialValues(existing: NodeWithEndShape): NodeWithEndShape {
    return { ...existing };
}

export function transformNodeWithEndValues(values: NodeWithEndShape, existing: NodeWithEndShape, isCreate: boolean) {
    return isCreate ? shapeNode.create(values) : shapeNode.update(existing, values);
}

function NodeWithEndForm({
    disabled,
    display,
    existing,
    isCreate,
    isEditing,
    isOpen,
    isReadLoading,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: NodeWithEndFormProps) {
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

    const [wasSuccessfulField] = useField<boolean>("end.wasSuccessful");

    const { handleCancel, handleCompleted } = useUpsertActions<NodeWithEndShape>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "Node",
        onAction: onClose,
        onCompleted: onCompleted as (data: NodeWithEndShape) => unknown,
        onDeleted: onDeleted as (data: NodeWithEndShape) => unknown,
        suppressSnack: true,
        ...props,
    });
    useSaveToCache({ isCacheOn: false, isCreate, values, objectId: values.id, objectType: "Node" });

    const onSubmit = useCallback(() => {
        handleCompleted(values);
    }, [handleCompleted, values]);

    const isLoading = useMemo(() => isReadLoading || props.isSubmitting, [isReadLoading, props.isSubmitting]);

    const topBarOptions = useMemo(function topBarOptionsMemo() {
        if (isCreate) return [];
        return [{
            Icon: DeleteIcon,
            label: t("Delete"),
            onClick: () => { onDeleted?.(existing as NodeWithEnd); },
        }];
    }, [existing, isCreate, onDeleted, t]);

    const nameProps = useMemo(function namePropsMemo() {
        return {
            language,
            fullWidth: true,
            multiline: false,
        } as const;
    }, [language]);

    const descriptionProps = useMemo(function descriptionPropsMemo() {
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
            id="node-with-end-crud-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display="dialog"
                onClose={onClose}
                title={firstString(getDisplay(values).title, t(isCreate ? "CreateNodeEnd" : "UpdateNodeEnd"))}
                options={topBarOptions}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={500}
            >
                <FormContainer marginBottom={0}>
                    <EditableTextCollapse
                        component="TranslatedTextInput"
                        isEditing={isEditing}
                        name="name"
                        props={nameProps}
                        title={t("Label")}
                    />
                    <EditableTextCollapse
                        component='TranslatedMarkdown'
                        isEditing={isEditing}
                        name="description"
                        props={descriptionProps}
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

export function NodeWithEndCrud({
    isEditing,
    isOpen,
    overrideObject,
    ...props
}: NodeWithEndCrudProps) {

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<NodeWithEndShape, NodeWithEndShape>({
        ...endpointsApi.findOne, // Won't be used. Need to pass an endpoint to useManagedObject
        isCreate: false,
        objectType: "Node",
        overrideObject: overrideObject as NodeWithEndShape,
        transform: (existing) => nodeWithEndInitialValues(existing as NodeWithEndShape),
    });

    async function validateValues(values: NodeWithEndShape) {
        return await validateFormValues(values, existing, false, transformNodeWithEndValues, nodeValidation);
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
                    <NodeWithEndForm
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
