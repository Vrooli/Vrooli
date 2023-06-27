import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { SubroutineForm, subroutineInitialValues, validateSubroutineValues } from "forms/SubroutineForm/SubroutineForm";
import { useContext, useMemo, useRef } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { SessionContext } from "utils/SessionContext";
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

    const { subroutine, numSubroutines } = useMemo(() => {
        if (!data?.node || !data?.routineItemId) return { subroutine: undefined, numSubroutines: 0 };
        const subroutine = data.node.routineList.items.find(r => r.id === data.routineItemId);
        return { subroutine, numSubroutines: data.node.routineList.items.length };
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
                    canUpdateRoutineVersion={canUpdate}
                    handleViewFull={handleViewFull}
                    isCreate={false}
                    isEditing={isEditing}
                    isOpen={true}
                    numSubroutines={numSubroutines}
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
