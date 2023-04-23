import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Formik } from "formik";
import { useContext, useMemo, useRef } from "react";
import { NodeEndForm, nodeEndInitialValues, validateNodeEndValues } from "../../../../forms/NodeEndForm/NodeEndForm";
import { SessionContext } from "../../../../utils/SessionContext";
import { LargeDialog } from "../../../dialogs/LargeDialog/LargeDialog";
import { TopBar } from "../../../navigation/TopBar/TopBar";
const titleId = "end-node-dialog-title";
export const NodeEndDialog = ({ handleClose, isEditing, isOpen, node, language, zIndex, }) => {
    const session = useContext(SessionContext);
    const formRef = useRef();
    const initialValues = useMemo(() => nodeEndInitialValues(session, node.routineVersion, node), [node, session]);
    return (_jsxs(LargeDialog, { id: "end-node-dialog", onClose: handleClose, isOpen: isOpen, titleId: titleId, zIndex: zIndex, children: [_jsx(TopBar, { display: "dialog", onClose: handleClose, titleData: { titleId, titleKey: isEditing ? "NodeEndEdit" : "NodeEndInfo" } }), _jsx(Formik, { enableReinitialize: true, initialValues: initialValues, onSubmit: (values) => {
                    handleClose(values);
                }, validate: async (values) => await validateNodeEndValues(values, node), children: (formik) => _jsx(NodeEndForm, { display: "dialog", isCreate: false, isEditing: isEditing, isLoading: false, isOpen: isOpen, onCancel: handleClose, ref: formRef, zIndex: zIndex, ...formik }) })] }));
};
//# sourceMappingURL=NodeEndDialog.js.map