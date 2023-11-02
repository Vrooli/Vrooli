import { DUMMY_ID, nodeTranslationValidation, NodeType, nodeValidation, orDefault, RoutineVersion, Session, uuid } from "@local/shared";
import { Checkbox, FormControlLabel, Tooltip } from "@mui/material";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { EditableTextCollapse } from "components/containers/EditableTextCollapse/EditableTextCollapse";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik, useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { NodeEndFormProps, NodeWithEndShape } from "forms/types";
import { useFormDialog } from "hooks/useConfirmBeforeLeave";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { forwardRef, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer } from "styles";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { validateAndGetYupErrors } from "utils/shape/general";
import { shapeNode } from "utils/shape/models/node";
import { NodeEndDialogProps } from "../types";

export const nodeEndInitialValues = (
    session: Session | undefined,
    routineVersion: RoutineVersion,
    existing?: NodeWithEndShape | null | undefined,
): NodeWithEndShape => {
    const id = uuid();
    return {
        __typename: "Node" as const,
        id,
        nodeType: NodeType.End,
        routineVersion,
        ...existing,
        end: {
            id: DUMMY_ID,
            __typename: "NodeEnd" as const,
            node: { __typename: "Node" as const, id },
            suggestedNextRoutineVersions: [],
            wasSuccessful: true,
            ...existing?.end,
        },
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
    const validationSchema = nodeValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" });
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
        validationSchema: nodeTranslationValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" }),
    });

    const [wasSuccessfulField] = useField<boolean>("end.wasSuccessful");

    return (
        <>
            <BaseForm
                display={display}
                isLoading={isLoading}
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
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
});

const titleId = "end-node-dialog-title";

export const NodeEndDialog = ({
    handleClose,
    isEditing,
    isOpen,
    node,
    language,
}: NodeEndDialogProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { formRef, handleClose: onClose } = useFormDialog({ handleCancel: handleClose });
    const initialValues = useMemo(() => nodeEndInitialValues(session, node.routineVersion, node), [node, session]);

    return (
        <LargeDialog
            id="end-node-dialog"
            onClose={onClose}
            isOpen={isOpen}
            titleId={titleId}
        >
            <TopBar
                display="dialog"
                onClose={onClose}
                title={t(isEditing ? "NodeEndEdit" : "NodeEndInfo")}
                titleId={titleId}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values) => {
                    handleClose(values as any);
                }}
                validate={async (values) => await validateNodeEndValues(values, node, false)}
            >
                {(formik) => <NodeEndForm
                    display="dialog"
                    isCreate={false}
                    isEditing={isEditing}
                    isLoading={false}
                    isOpen={isOpen}
                    onCancel={handleClose}
                    onClose={onClose}
                    ref={formRef}
                    {...formik}
                />}
            </Formik>
        </LargeDialog>
    );
};
