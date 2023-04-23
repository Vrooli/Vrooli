import { NodeRoutineListItem } from "@local/consts";
import { Formik } from "formik";
import { useContext, useMemo, useRef } from "react";
import { BaseFormRef } from "../../../forms/BaseForm/BaseForm";
import { SubroutineForm, subroutineInitialValues, validateSubroutineValues } from "../../../forms/SubroutineForm/SubroutineForm";
import { getCurrentUser } from "../../../utils/authentication/session";
import { SessionContext } from "../../../utils/SessionContext";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { SubroutineInfoDialogProps } from "../types";

/**
 * Drawer to display a routine list item's info on the build page. 
 * Swipes up from bottom of screen
 */
export const SubroutineInfoDialog = ({
    data,
    defaultLanguage,
    handleUpdate,
    handleReorder,
    handleViewFull,
    isEditing,
    open,
    onClose,
    zIndex,
}: SubroutineInfoDialogProps) => {
    const session = useContext(SessionContext);

    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    const subroutine = useMemo<NodeRoutineListItem | undefined>(() => {
        if (!data?.node || !data?.routineItemId) return undefined;
        return data.node.routineList.items.find(r => r.id === data.routineItemId);
    }, [data]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => subroutineInitialValues(session, subroutine), [subroutine, session]);

    const canUpdate = useMemo<boolean>(() => isEditing && (subroutine?.routineVersion?.root?.isInternal || subroutine?.routineVersion?.root?.owner?.id === userId || subroutine?.routineVersion?.you?.canUpdate === true), [isEditing, subroutine?.routineVersion?.root?.isInternal, subroutine?.routineVersion?.root?.owner?.id, subroutine?.routineVersion?.you?.canUpdate, userId]);

    return (
        <LargeDialog
            id="subroutine-dialog"
            onClose={onClose}
            isOpen={open}
            titleId={""}
            zIndex={zIndex}
        >
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!data || !subroutine) return;
                    // Check if subroutine index has changed
                    const originalIndex = subroutine.index;
                    // Update the subroutine
                    handleUpdate(values as any);
                    // If the index has changed, reorder the subroutine
                    originalIndex !== values.index && handleReorder(data.node.id, originalIndex, values.index);
                }}
                validate={async (values) => await validateSubroutineValues(values, subroutine)}
            >
                {(formik) => <SubroutineForm
                    canUpdate={canUpdate}
                    handleViewFull={handleViewFull}
                    isCreate={true}
                    isEditing={isEditing}
                    isOpen={true}
                    onCancel={onClose}
                    ref={formRef}
                    versions={[]}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </LargeDialog>
    );
};
