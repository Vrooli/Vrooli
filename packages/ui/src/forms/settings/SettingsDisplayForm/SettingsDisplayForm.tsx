import { Stack } from "@mui/material";
import { GridSubmitButtons } from "../../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { FocusModeSelector } from "../../../components/inputs/FocusModeSelector/FocusModeSelector";
import { LanguageSelector } from "../../../components/inputs/LanguageSelector/LanguageSelector";
import { LeftHandedCheckbox } from "../../../components/inputs/LeftHandedCheckbox/LeftHandedCheckbox";
import { TextSizeButtons } from "../../../components/inputs/TextSizeButtons/TextSizeButtons";
import { ThemeSwitch } from "../../../components/inputs/ThemeSwitch/ThemeSwitch";
import { Subheader } from "../../../components/text/Subheader/Subheader";
import { BaseForm } from "../../../forms/BaseForm/BaseForm";
import { useTranslation } from "react-i18next";
import { SettingsDisplayFormProps } from "../types";

export const SettingsDisplayForm = ({
    display,
    dirty,
    isLoading,
    onCancel,
    values,
    ...props
}: SettingsDisplayFormProps) => {
    const { t } = useTranslation();

    return (
        <BaseForm
            dirty={dirty}
            isLoading={isLoading}
            style={{
                width: { xs: "100%", md: "min(100%, 700px)" },
                margin: "auto",
                display: "block",
            }}
        >
            <Subheader
                help={t("DisplayAccountHelp")}
                title={t("DisplayAccount")} />
            <Stack direction="column" spacing={2} p={1}>
                <LanguageSelector />
                <FocusModeSelector />
            </Stack>
            <Subheader
                help={t("DisplayDeviceHelp")}
                title={t("DisplayDevice")} />
            <Stack direction="column" spacing={2} p={1}>
                <ThemeSwitch />
                <TextSizeButtons />
                <LeftHandedCheckbox />
            </Stack>
            <GridSubmitButtons
                display={display}
                errors={props.errors}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </BaseForm>
    );
};
