import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { LayoutList, Terminal } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { strings } from "../../consts/strings";
import { Button } from "../ui/button";
import HeaderColorPicker from "../appearance/HeaderColorPicker";
import ThemePicker from "../appearance/ThemePicker";
import FontSizeStepper from "../appearance/FontSizeStepper";
import { SettingsCard, SettingsRow, SettingsSectionIntro } from "./primitives";
export default function NewPaneDefaultsSection() {
    const { t } = useTranslation();
    const defaultHeaderColor = useWorkspaceStore((state) => state.defaultHeaderColor);
    const defaultThemeId = useWorkspaceStore((state) => state.defaultThemeId);
    const defaultFontSize = useWorkspaceStore((state) => state.defaultFontSize);
    const plusButtonBehavior = useWorkspaceStore((state) => state.plusButtonBehavior);
    const setDefaultHeaderColor = useWorkspaceStore((state) => state.setDefaultHeaderColor);
    const setDefaultThemeId = useWorkspaceStore((state) => state.setDefaultThemeId);
    const setDefaultFontSize = useWorkspaceStore((state) => state.setDefaultFontSize);
    const setPlusButtonBehavior = useWorkspaceStore((state) => state.setPlusButtonBehavior);
    return (_jsxs("div", { className: "space-y-4", children: [_jsx(SettingsSectionIntro, { eyebrow: t(strings.settings.newPaneDefaultsSection.eyebrow), title: t(strings.settings.newPaneDefaultsSection.title), description: t(strings.settings.newPaneDefaultsSection.description) }), _jsxs(SettingsCard, { className: "space-y-5", children: [_jsx(HeaderColorPicker, { currentColor: defaultHeaderColor, onSelectColor: setDefaultHeaderColor, testIdPrefix: "defaults" }), _jsx(ThemePicker, { currentThemeId: defaultThemeId, onSelectTheme: setDefaultThemeId, testIdPrefix: "defaults" }), _jsx(FontSizeStepper, { currentSize: defaultFontSize, onChangeSize: setDefaultFontSize, testIdPrefix: "defaults" }), _jsx(SettingsRow, { label: t(strings.settings.newPaneDefaultsSection.plusButtonLabel), hint: t(strings.settings.newPaneDefaultsSection.plusButtonHint), control: (_jsxs("div", { className: "flex items-center gap-2", children: [_jsxs(Button, { "data-testid": "plus-behavior-launcher", variant: plusButtonBehavior === "launcher" ? "default" : "outline", size: "sm", className: "h-8 px-3", onClick: () => setPlusButtonBehavior("launcher"), children: [_jsx(LayoutList, { className: "me-1 h-3.5 w-3.5" }), t(strings.settings.newPaneDefaultsSection.launcher)] }), _jsxs(Button, { "data-testid": "plus-behavior-new-terminal", variant: plusButtonBehavior === "new-terminal" ? "default" : "outline", size: "sm", className: "h-8 px-3", onClick: () => setPlusButtonBehavior("new-terminal"), children: [_jsx(Terminal, { className: "me-1 h-3.5 w-3.5" }), t(strings.settings.newPaneDefaultsSection.emptyTerminal)] })] })) })] })] }));
}
