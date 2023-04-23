import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { Formik } from "formik";
import { useContext, useEffect, useMemo, useRef } from "react";
import { routineVersionCreate } from "../../../api/generated/endpoints/routineVersion_create";
import { routineVersionFindOne } from "../../../api/generated/endpoints/routineVersion_findOne";
import { routineVersionUpdate } from "../../../api/generated/endpoints/routineVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { RoutineForm, routineInitialValues, transformRoutineValues, validateRoutineValues } from "../../../forms/RoutineForm/RoutineForm";
import { useUpsertActions } from "../../../utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "../../../utils/navigation/urlTools";
import { PubSub } from "../../../utils/pubsub";
import { SessionContext } from "../../../utils/SessionContext";
export const RoutineUpsert = ({ display = "page", isCreate, isSubroutine = false, onCancel, onCompleted, zIndex = 200, }) => {
    const session = useContext(SessionContext);
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery(routineVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }); }, [getData, id]);
    const formRef = useRef();
    const initialValues = useMemo(() => routineInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation(routineVersionCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation(routineVersionUpdate);
    const mutation = isCreate ? create : update;
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: handleCancel, titleData: {
                    titleKey: isCreate ? "CreateRoutine" : "UpdateRoutine",
                } }), _jsx(Formik, { enableReinitialize: true, initialValues: initialValues, onSubmit: (values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    mutationWrapper({
                        mutation,
                        input: transformRoutineValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }, validate: async (values) => await validateRoutineValues(values, existing), children: (formik) => _jsx(RoutineForm, { display: display, isCreate: isCreate, isLoading: isCreateLoading || isReadLoading || isUpdateLoading, isOpen: true, isSubroutine: isSubroutine, onCancel: handleCancel, ref: formRef, versions: existing?.root?.versions?.map(v => v.versionLabel) ?? [], zIndex: zIndex, ...formik }) })] }));
};
//# sourceMappingURL=RoutineUpsert.js.map