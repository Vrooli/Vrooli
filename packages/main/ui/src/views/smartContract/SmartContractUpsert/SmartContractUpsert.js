import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { Formik } from "formik";
import { useContext, useEffect, useMemo, useRef } from "react";
import { mutationWrapper } from "../../../api";
import { smartContractVersionFindOne } from "../../../api/generated/endpoints/smartContractVersion_findOne";
import { smartContractCreate } from "../../../api/generated/endpoints/smartContract_create";
import { smartContractUpdate } from "../../../api/generated/endpoints/smartContract_update";
import { useCustomLazyQuery, useCustomMutation } from "../../../api/hooks";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { SmartContractForm, smartContractInitialValues, transformSmartContractValues, validateSmartContractValues } from "../../../forms/SmartContractForm/SmartContractForm";
import { useUpsertActions } from "../../../utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "../../../utils/navigation/urlTools";
import { PubSub } from "../../../utils/pubsub";
import { SessionContext } from "../../../utils/SessionContext";
export const SmartContractUpsert = ({ display = "page", isCreate, onCancel, onCompleted, zIndex = 200, }) => {
    const session = useContext(SessionContext);
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery(smartContractVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }); }, [getData, id]);
    const formRef = useRef();
    const initialValues = useMemo(() => smartContractInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation(smartContractCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation(smartContractUpdate);
    const mutation = isCreate ? create : update;
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: handleCancel, titleData: {
                    titleKey: isCreate ? "CreateSmartContract" : "UpdateSmartContract",
                } }), _jsx(Formik, { enableReinitialize: true, initialValues: initialValues, onSubmit: (values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    mutationWrapper({
                        mutation,
                        input: transformSmartContractValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }, validate: async (values) => await validateSmartContractValues(values, existing), children: (formik) => _jsx(SmartContractForm, { display: display, isCreate: isCreate, isLoading: isCreateLoading || isReadLoading || isUpdateLoading, isOpen: true, onCancel: handleCancel, ref: formRef, versions: existing?.root?.versions?.map(v => v.versionLabel) ?? [], zIndex: zIndex, ...formik }) })] }));
};
//# sourceMappingURL=SmartContractUpsert.js.map