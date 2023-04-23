import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { Formik } from "formik";
import { useContext, useEffect, useMemo, useRef } from "react";
import { standardVersionCreate } from "../../../api/generated/endpoints/standardVersion_create";
import { standardVersionFindOne } from "../../../api/generated/endpoints/standardVersion_findOne";
import { standardVersionUpdate } from "../../../api/generated/endpoints/standardVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { StandardForm, standardInitialValues, transformStandardValues, validateStandardValues } from "../../../forms/StandardForm/StandardForm";
import { useUpsertActions } from "../../../utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "../../../utils/navigation/urlTools";
import { PubSub } from "../../../utils/pubsub";
import { SessionContext } from "../../../utils/SessionContext";
export const StandardUpsert = ({ display = "page", isCreate, onCancel, onCompleted, zIndex = 200, }) => {
    const session = useContext(SessionContext);
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery(standardVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }); }, [getData, id]);
    const formRef = useRef();
    const initialValues = useMemo(() => standardInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation(standardVersionCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation(standardVersionUpdate);
    const mutation = isCreate ? create : update;
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: handleCancel, titleData: {
                    titleKey: isCreate ? "CreateStandard" : "UpdateStandard",
                } }), _jsx(Formik, { enableReinitialize: true, initialValues: initialValues, onSubmit: (values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    mutationWrapper({
                        mutation,
                        input: transformStandardValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }, validate: async (values) => await validateStandardValues(values, existing), children: (formik) => _jsx(StandardForm, { display: display, isCreate: isCreate, isLoading: isCreateLoading || isReadLoading || isUpdateLoading, isOpen: true, onCancel: handleCancel, ref: formRef, versions: existing?.root?.versions?.map(v => v.versionLabel) ?? [], zIndex: zIndex, ...formik }) })] }));
};
//# sourceMappingURL=StandardUpsert.js.map