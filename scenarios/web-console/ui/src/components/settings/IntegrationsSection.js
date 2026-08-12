import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
import IntegrationsPanel from "../IntegrationsPanel";
import { SettingsCard, SettingsSectionIntro } from "./primitives";
export default function IntegrationsSection({ open }) {
    const { t } = useTranslation();
    return (_jsxs("div", { className: "space-y-4", children: [_jsx(SettingsSectionIntro, { eyebrow: t(strings.settings.integrationsSection.eyebrow), title: t(strings.settings.integrationsSection.title), description: t(strings.settings.integrationsSection.description) }), _jsx(SettingsCard, { children: _jsx(IntegrationsPanel, { open: open }) })] }));
}
