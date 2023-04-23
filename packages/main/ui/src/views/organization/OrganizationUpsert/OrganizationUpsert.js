import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { Formik } from "formik";
import { useContext, useEffect, useMemo, useRef } from "react";
import { organizationCreate } from "../../../api/generated/endpoints/organization_create";
import { organizationFindOne } from "../../../api/generated/endpoints/organization_findOne";
import { organizationUpdate } from "../../../api/generated/endpoints/organization_update";
import { useCustomLazyQuery, useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { OrganizationForm, organizationInitialValues, transformOrganizationValues, validateOrganizationValues } from "../../../forms/OrganizationForm/OrganizationForm";
import { useUpsertActions } from "../../../utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "../../../utils/navigation/urlTools";
import { PubSub } from "../../../utils/pubsub";
import { SessionContext } from "../../../utils/SessionContext";
export const OrganizationUpsert = ({ display = "page", isCreate, onCancel, onCompleted, zIndex = 200, }) => {
    const session = useContext(SessionContext);
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery(organizationFindOne);
    useEffect(() => { id && getData({ variables: { id } }); }, [getData, id]);
    const formRef = useRef();
    const initialValues = useMemo(() => organizationInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation(organizationCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation(organizationUpdate);
    const mutation = isCreate ? create : update;
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: handleCancel, titleData: {
                    titleKey: isCreate ? "CreateOrganization" : "UpdateOrganization",
                } }), _jsx(Formik, { enableReinitialize: true, initialValues: initialValues, onSubmit: (values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    mutationWrapper({
                        mutation,
                        input: transformOrganizationValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }, validate: async (values) => await validateOrganizationValues(values, existing), children: (formik) => _jsx(OrganizationForm, { display: display, isCreate: isCreate, isLoading: isCreateLoading || isReadLoading || isUpdateLoading, isOpen: true, onCancel: handleCancel, ref: formRef, zIndex: zIndex, ...formik }) })] }));
};
//# sourceMappingURL=OrganizationUpsert.js.map