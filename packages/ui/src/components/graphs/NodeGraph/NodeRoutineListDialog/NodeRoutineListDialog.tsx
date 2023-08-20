import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { NodeRoutineListForm, nodeRoutineListInitialValues, validateNodeRoutineListValues } from "forms/NodeRoutineListForm/NodeRoutineListForm";
import { useContext, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { NodeRoutineListDialogProps } from "../types";

const titleId = "routine-list-node-dialog-title";

export const NodeRoutineListDialog = ({
    handleClose,
    isEditing,
    isOpen,
    node,
    language,
    zIndex,
}: NodeRoutineListDialogProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => nodeRoutineListInitialValues(session, node?.routineVersion as any, node), [node, session]);

    return (
        <LargeDialog
            id="routine-list-node-dialog"
            onClose={() => { handleClose(); }}
            isOpen={isOpen}
            titleId={titleId}
            zIndex={zIndex}
        >
            <TopBar
                display="dialog"
                onClose={handleClose}
                title={t(isEditing ? "NodeRoutineListEdit" : "NodeRoutineListInfo")}
                titleId={titleId}
                zIndex={zIndex}
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
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </LargeDialog>
    );
};
