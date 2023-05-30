import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { NodeRoutineListForm, nodeRoutineListInitialValues, validateNodeRoutineListValues } from "forms/NodeRoutineListForm/NodeRoutineListForm";
import { useContext, useMemo, useRef } from "react";
import { SessionContext } from "utils/SessionContext";
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
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => nodeRoutineListInitialValues(session, node?.routineVersion as any, node), [node, session]);

    return (
        <LargeDialog
            id="routine-list-node-dialog"
            onClose={handleClose}
            isOpen={isOpen}
            titleId={titleId}
            zIndex={zIndex}
        >
            <TopBar
                display="dialog"
                onClose={handleClose}
                titleData={{ titleId, titleKey: isEditing ? "NodeRoutineListEdit" : "NodeRoutineListInfo" }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values) => {
                    handleClose(values as any);
                }}
                validate={async (values) => await validateNodeRoutineListValues(values, node ?? undefined)}
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
