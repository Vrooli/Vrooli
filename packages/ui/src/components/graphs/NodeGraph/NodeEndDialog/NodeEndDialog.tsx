import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { NodeEndForm, nodeEndInitialValues, validateNodeEndValues } from "forms/NodeEndForm/NodeEndForm";
import { useFormDialog } from "hooks/useFormDialog";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { NodeEndDialogProps } from "../types";

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
