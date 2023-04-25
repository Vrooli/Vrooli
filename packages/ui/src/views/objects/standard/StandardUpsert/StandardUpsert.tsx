import { FindVersionInput, StandardVersion, StandardVersionCreateInput, StandardVersionUpdateInput } from "@local/shared";
import { standardVersionCreate } from "api/generated/endpoints/standardVersion_create";
import { standardVersionFindOne } from "api/generated/endpoints/standardVersion_findOne";
import { standardVersionUpdate } from "api/generated/endpoints/standardVersion_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { mutationWrapper } from "api/utils";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { StandardForm, standardInitialValues, transformStandardValues, validateStandardValues } from "forms/StandardForm/StandardForm";
import { useContext, useEffect, useMemo, useRef } from "react";
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
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<StandardVersion, FindVersionInput>(standardVersionFindOne);
    useEffect(() => { id && getData({ variables: { id } }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => standardInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<StandardVersion>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation<StandardVersion, StandardVersionCreateInput>(standardVersionCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation<StandardVersion, StandardVersionUpdateInput>(standardVersionUpdate);
    const mutation = isCreate ? create : update;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: isCreate ? "CreateStandard" : "UpdateStandard",
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    mutationWrapper<StandardVersion, StandardVersionUpdateInput>({
                        mutation,
                        input: transformStandardValues(values, existing),
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
