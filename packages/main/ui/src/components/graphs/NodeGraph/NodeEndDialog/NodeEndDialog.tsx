import { Formik } from "formik";
import { useContext, useMemo, useRef } from "react";
import { BaseFormRef } from "../../../../forms/BaseForm/BaseForm";
import { NodeEndForm, nodeEndInitialValues, validateNodeEndValues } from "../../../../forms/NodeEndForm/NodeEndForm";
import { SessionContext } from "../../../../utils/SessionContext";
import { LargeDialog } from "../../../dialogs/LargeDialog/LargeDialog";
import { TopBar } from "../../../navigation/TopBar/TopBar";
import { NodeEndDialogProps } from "../types";

const titleId = "end-node-dialog-title";

export const NodeEndDialog = ({
    handleClose,
    isEditing,
    isOpen,
    node,
    language,
    zIndex,
}: NodeEndDialogProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => nodeEndInitialValues(session, node.routineVersion, node), [node, session]);

    return (
        <LargeDialog
            id="end-node-dialog"
            onClose={handleClose}
            isOpen={isOpen}
            titleId={titleId}
            zIndex={zIndex}
        >
            <TopBar
                display="dialog"
                onClose={handleClose}
                titleData={{ titleId, titleKey: isEditing ? "NodeEndEdit" : "NodeEndInfo" }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values) => {
                    handleClose(values as any);
                }}
                validate={async (values) => await validateNodeEndValues(values, node)}
            >
                {(formik) => <NodeEndForm
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
