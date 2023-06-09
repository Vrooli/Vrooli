import { endpointGetStandardVersion, endpointPostStandardVersion, endpointPutStandardVersion, FindVersionInput, StandardVersion, StandardVersionCreateInput, StandardVersionUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { StandardForm, standardInitialValues, transformStandardValues, validateStandardValues } from "forms/StandardForm/StandardForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { StandardUpsertProps } from "../types";

export const StandardUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex = 200,
}: StandardUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useLazyFetch<FindVersionInput, StandardVersion>(endpointGetStandardVersion);
    useEffect(() => { id && getData({ id }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => standardInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<StandardVersion>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<StandardVersionCreateInput, StandardVersion>(endpointPostStandardVersion);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<StandardVersionUpdateInput, StandardVersion>(endpointPutStandardVersion);
    const fetch = (isCreate ? create : update) as MakeLazyRequest<StandardVersionCreateInput | StandardVersionUpdateInput, StandardVersion>;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateStandard" : "UpdateStandard")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<StandardVersionCreateInput | StandardVersionUpdateInput, StandardVersion>({
                        fetch,
                        inputs: transformStandardValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateStandardValues(values, existing)}
            >
                {(formik) => <StandardForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    versions={existing?.root?.versions?.map(v => v.versionLabel) ?? []}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    );
};
