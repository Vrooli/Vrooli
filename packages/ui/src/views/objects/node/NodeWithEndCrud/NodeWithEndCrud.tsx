import { endpointGetApi, nodeTranslationValidation, nodeValidation, noopSubmit } from "@local/shared";
import { Checkbox, FormControlLabel, Tooltip } from "@mui/material";
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
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { shapeNode } from "utils/shape/models/node";
import { validateFormValues } from "utils/validateFormValues";
import { NodeWithEnd, NodeWithEndCrudProps, NodeWithEndFormProps, NodeWithEndShape } from "../types";

export function nodeWithEndInitialValues(existing: NodeWithEndShape): NodeWithEndShape {
    return { ...existing };
}

export function transformNodeWithEndValues(values: NodeWithEndShape, existing: NodeWithEndShape, isCreate: boolean) {
    return isCreate ? shapeNode.create(values) : shapeNode.update(existing, values);
}

function NodeWithEndForm({
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
            id="node-with-end-crud-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display="dialog"
                onClose={onClose}
                title={firstString(getDisplay(values).title, t(isCreate ? "CreateNodeEnd" : "UpdateNodeEnd"))}
                options={!isCreate ? [{
                    Icon: DeleteIcon,
                    label: t("Delete"),
                    onClick: () => { onDeleted?.(existing as NodeWithEnd); },
                }] : []}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={500}
            >
                <FormContainer sx={{ marginBottom: 8 }}>
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

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useObjectFromUrl<NodeWithEndShape, NodeWithEndShape>({
        ...endpointGetApi, // Won't be used. Need to pass an endpoint to useObjectFromUrl
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
