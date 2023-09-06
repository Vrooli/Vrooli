import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { NodeRoutineListForm, nodeRoutineListInitialValues, validateNodeRoutineListValues } from "forms/NodeRoutineListForm/NodeRoutineListForm";
import { useFormDialog } from "hooks/useFormDialog";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { NodeRoutineListDialogProps } from "../types";

const titleId = "routine-list-node-dialog-title";

export const NodeRoutineListDialog = ({
    handleClose,
    isEditing,
    isOpen,
    node,
    language,
}: NodeRoutineListDialogProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { formRef, handleClose: onClose } = useFormDialog({ handleCancel: handleClose });
    const initialValues = useMemo(() => nodeRoutineListInitialValues(session, node?.routineVersion as any, node), [node, session]);

    return (
        <LargeDialog
            id="routine-list-node-dialog"
            onClose={onClose}
            isOpen={isOpen}
            titleId={titleId}
        >
            <TopBar
                display="dialog"
                onClose={onClose}
                title={t(isEditing ? "NodeRoutineListEdit" : "NodeRoutineListInfo")}
                titleId={titleId}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values) => {
                    handleClose(values);
                }}
                validate={async (values) => await validateNodeRoutineListValues(values, initialValues, false)}
            >
                {(formik) => <NodeRoutineListForm
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
