import { jsx as _jsx } from "react/jsx-runtime";
import { Formik } from "formik";
import { useContext, useMemo, useRef } from "react";
import { SubroutineForm, subroutineInitialValues, validateSubroutineValues } from "../../../forms/SubroutineForm/SubroutineForm";
import { getCurrentUser } from "../../../utils/authentication/session";
import { SessionContext } from "../../../utils/SessionContext";
import { LargeDialog } from "../LargeDialog/LargeDialog";
export const SubroutineInfoDialog = ({ data, defaultLanguage, handleUpdate, handleReorder, handleViewFull, isEditing, open, onClose, zIndex, }) => {
    const session = useContext(SessionContext);
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);
    const subroutine = useMemo(() => {
        if (!data?.node || !data?.routineItemId)
            return undefined;
        return data.node.routineList.items.find(r => r.id === data.routineItemId);
    }, [data]);
    const formRef = useRef();
    const initialValues = useMemo(() => subroutineInitialValues(session, subroutine), [subroutine, session]);
    const canUpdate = useMemo(() => isEditing && (subroutine?.routineVersion?.root?.isInternal || subroutine?.routineVersion?.root?.owner?.id === userId || subroutine?.routineVersion?.you?.canUpdate === true), [isEditing, subroutine?.routineVersion?.root?.isInternal, subroutine?.routineVersion?.root?.owner?.id, subroutine?.routineVersion?.you?.canUpdate, userId]);
    return (_jsx(LargeDialog, { id: "subroutine-dialog", onClose: onClose, isOpen: open, titleId: "", zIndex: zIndex, children: _jsx(Formik, { enableReinitialize: true, initialValues: initialValues, onSubmit: (values, helpers) => {
                if (!data || !subroutine)
                    return;
                const originalIndex = subroutine.index;
                handleUpdate(values);
                originalIndex !== values.index && handleReorder(data.node.id, originalIndex, values.index);
            }, validate: async (values) => await validateSubroutineValues(values, subroutine), children: (formik) => _jsx(SubroutineForm, { canUpdate: canUpdate, handleViewFull: handleViewFull, isCreate: true, isEditing: isEditing, isOpen: true, onCancel: onClose, ref: formRef, versions: [], zIndex: zIndex, ...formik }) }) }));
};
//# sourceMappingURL=SubroutineInfoDialog.js.map