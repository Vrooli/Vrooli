import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { Formik } from "formik";
import { useContext, useEffect, useMemo, useRef } from "react";
import { mutationWrapper } from "../../../api";
import { apiVersionCreate } from "../../../api/generated/endpoints/apiVersion_create";
import { apiVersionFindOne } from "../../../api/generated/endpoints/apiVersion_findOne";
import { apiVersionUpdate } from "../../../api/generated/endpoints/apiVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "../../../api/hooks";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { ApiForm, apiInitialValues, transformApiValues, validateApiValues } from "../../../forms/ApiForm/ApiForm";
import { useUpsertActions } from "../../../utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "../../../utils/navigation/urlTools";
import { PubSub } from "../../../utils/pubsub";
import { SessionContext } from "../../../utils/SessionContext";
export const ApiUpsert = ({ display = "page", isCreate, onCancel, onCompleted, zIndex = 200, }) => {
    const session = useContext(SessionContext);
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery(apiVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }); }, [getData, id]);
    const formRef = useRef();
    const initialValues = useMemo(() => apiInitialValues(session), [session]);
    const { handleCancel, handleCompleted } = useUpsertActions(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation(apiVersionCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation(apiVersionUpdate);
    const mutation = isCreate ? create : update;
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: handleCancel, titleData: {
                    titleKey: isCreate ? "CreateApi" : "UpdateApi",
                } }), _jsx(Formik, { enableReinitialize: true, initialValues: initialValues, onSubmit: (values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    mutationWrapper({
                        mutation,
                        input: transformApiValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }, validate: async (values) => await validateApiValues(values, existing), children: (formik) => _jsx(ApiForm, { display: display, isCreate: isCreate, isLoading: isCreateLoading || isReadLoading || isUpdateLoading, isOpen: true, onCancel: handleCancel, ref: formRef, versions: [], zIndex: zIndex, ...formik }) })] }));
};
//# sourceMappingURL=ApiUpsert.js.map