import { Stack } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { FocusModeSelector } from "components/inputs/FocusModeSelector/FocusModeSelector";
import { LanguageSelector } from "components/inputs/LanguageSelector/LanguageSelector";
import { LeftHandedCheckbox } from "components/inputs/LeftHandedCheckbox/LeftHandedCheckbox";
import { TextSizeButtons } from "components/inputs/TextSizeButtons/TextSizeButtons";
import { ThemeSwitch } from "components/inputs/ThemeSwitch/ThemeSwitch";
import { Title } from "components/text/Title/Title";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useTranslation } from "react-i18next";
import { SettingsDisplayFormProps } from "../types";

export const SettingsDisplayForm = ({
    display,
    dirty,
    isLoading,
    onCancel,
    values,
    zIndex,
    ...props
}: SettingsDisplayFormProps) => {
    const { t } = useTranslation();

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
            >
                <Title
                    help={t("DisplayAccountHelp")}
                    title={t("DisplayAccount")}
                    variant="subheader"
                    zIndex={zIndex}
                />
                <Stack direction="column" spacing={2} p={1}>
                    <LanguageSelector />
                    <FocusModeSelector />
                </Stack>
                <Title
                    help={t("DisplayDeviceHelp")}
                    title={t("DisplayDevice")}
                    variant="subheader"
                    zIndex={zIndex}
                />
                <Stack direction="column" spacing={2} p={1}>
                    <ThemeSwitch />
                    <TextSizeButtons />
                    <LeftHandedCheckbox />
                </Stack>
            </BaseForm>
            <GridSubmitButtons
                display={display}
                errors={props.errors}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
                zIndex={zIndex}
            />
        </>
    );
};
