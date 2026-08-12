import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useCallback, useEffect, useState } from "react";
import { AlertCircle, Plus, Save, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "../ui/button";
import { SettingsCard, SettingsSectionIntro } from "./primitives";
import { shortcutsClient } from "../../api/shortcuts";
import { strings } from "../../consts/strings";
import { toErrorInfo } from "../../lib/errors";
function ShortcutEditor({ profile, onSave, onDelete, }) {
    const { t } = useTranslation();
    const [entries, setEntries] = useState(profile.shortcuts.map((s) => ({ label: s.label, command: s.command, description: s.description })));
    const [name, setName] = useState(profile.name);
    const [dirty, setDirty] = useState(false);
    const updateEntry = (index, field, value) => {
        setEntries((current) => current.map((entry, idx) => (idx === index ? { ...entry, [field]: value } : entry)));
        setDirty(true);
    };
    const addEntry = () => {
        setEntries((current) => [...current, { label: "", command: "", description: "" }]);
        setDirty(true);
    };
    const removeEntry = (index) => {
        setEntries((current) => current.filter((_, idx) => idx !== index));
        setDirty(true);
    };
    return (_jsxs("div", { "data-testid": `shortcut-profile-${profile.id}`, className: "rounded-xl border border-wc-default bg-wc-surface-base/70 p-3", children: [_jsxs("div", { className: "mb-3 flex items-center justify-between gap-2", children: [_jsx("input", { "data-testid": `profile-name-${profile.id}`, className: "min-w-0 flex-1 border-b border-transparent bg-transparent text-sm font-medium text-wc-text-primary outline-none focus:border-wc-accent", value: name, onChange: (event) => {
                            setName(event.target.value);
                            setDirty(true);
                        } }), _jsxs("div", { className: "flex items-center gap-1", children: [dirty && (_jsx(Button, { "data-testid": `profile-save-${profile.id}`, variant: "ghost", size: "icon", className: "h-7 w-7", onClick: () => {
                                    onSave({ id: profile.id, scope: profile.scope, name, shortcuts: entries });
                                    setDirty(false);
                                }, title: t(strings.settings.shortcutsSection.saveChanges), children: _jsx(Save, { className: "h-3.5 w-3.5 text-green-400" }) })), _jsx(Button, { "data-testid": `profile-delete-${profile.id}`, variant: "ghost", size: "icon", className: "h-7 w-7", onClick: () => onDelete(profile.id), title: t(strings.settings.shortcutsSection.deleteProfile), children: _jsx(Trash2, { className: "h-3.5 w-3.5 text-wc-text-faint hover:text-wc-error-detail" }) })] })] }), _jsx("div", { className: "space-y-2", children: entries.map((entry, index) => (_jsxs("div", { className: "flex items-center gap-2", children: [_jsx("input", { "data-testid": `entry-label-${profile.id}-${index}`, placeholder: t(strings.settings.shortcutsSection.labelPlaceholder), className: "min-w-0 flex-1 rounded-lg border border-wc-default bg-wc-surface-input px-2 py-1 text-xs text-wc-text-primary outline-none focus:border-wc-accent", value: entry.label, onChange: (event) => updateEntry(index, "label", event.target.value) }), _jsx("input", { "data-testid": `entry-command-${profile.id}-${index}`, placeholder: t(strings.settings.shortcutsSection.commandPlaceholder), className: "min-w-0 flex-[1.5] rounded-lg border border-wc-default bg-wc-surface-input px-2 py-1 font-mono text-xs text-wc-text-primary outline-none focus:border-wc-accent", value: entry.command, onChange: (event) => updateEntry(index, "command", event.target.value) }), _jsx(Button, { variant: "ghost", size: "icon", className: "h-7 w-7 shrink-0", onClick: () => removeEntry(index), title: t(strings.settings.shortcutsSection.removeShortcut), children: _jsx(Trash2, { className: "h-3 w-3 text-wc-text-faint" }) })] }, `${profile.id}-${index}`))) }), _jsxs(Button, { variant: "ghost", size: "sm", className: "mt-2 text-xs text-wc-text-faint", onClick: addEntry, children: [_jsx(Plus, { className: "me-1 h-3 w-3" }), t(strings.settings.shortcutsSection.addShortcut)] })] }));
}
export default function ShortcutProfilesSection() {
    const { t } = useTranslation();
    const [profiles, setProfiles] = useState([]);
    const [profileError, setProfileError] = useState(null);
    const [profileLoading, setProfileLoading] = useState(true);
    const loadProfiles = useCallback(async (signal) => {
        setProfileLoading(true);
        try {
            const resp = await shortcutsClient.listProfiles({});
            if (signal?.cancelled)
                return;
            setProfiles(resp.profiles);
            setProfileError(null);
        }
        catch (error) {
            if (signal?.cancelled)
                return;
            setProfileError(toErrorInfo(error).message);
        }
        finally {
            if (!signal?.cancelled) {
                setProfileLoading(false);
            }
        }
    }, []);
    useEffect(() => {
        const signal = { cancelled: false };
        void loadProfiles(signal);
        return () => {
            signal.cancelled = true;
        };
    }, [loadProfiles]);
    const handleSaveProfile = useCallback(async (draft) => {
        try {
            const resp = await shortcutsClient.upsertProfile(draft);
            if (!resp.profile) {
                throw new Error("upsertProfile: missing profile in response");
            }
            const updated = resp.profile;
            setProfiles((current) => current.map((item) => (item.id === updated.id ? updated : item)));
        }
        catch (error) {
            setProfileError(toErrorInfo(error).message);
            void loadProfiles();
        }
    }, [loadProfiles]);
    const handleDeleteProfile = useCallback(async (id) => {
        try {
            await shortcutsClient.deleteProfile({ id });
            setProfiles((current) => current.filter((item) => item.id !== id));
        }
        catch (error) {
            setProfileError(toErrorInfo(error).message);
            void loadProfiles();
        }
    }, [loadProfiles]);
    const handleCreateProfile = useCallback(async () => {
        try {
            const resp = await shortcutsClient.upsertProfile({
                id: `profile-${Date.now()}`,
                scope: "workspace",
                name: t(strings.settings.shortcutsSection.newProfileName),
                shortcuts: [{ label: t(strings.settings.shortcutsSection.defaultShortcutLabel), command: "ls -la", description: "" }],
            });
            if (!resp.profile) {
                throw new Error("upsertProfile: missing profile in response");
            }
            const newProfile = resp.profile;
            setProfiles((current) => [...current, newProfile]);
        }
        catch (error) {
            setProfileError(toErrorInfo(error).message);
        }
    }, [t]);
    return (_jsxs("div", { className: "space-y-4", children: [_jsx(SettingsSectionIntro, { eyebrow: t(strings.settings.shortcutsSection.eyebrow), title: t(strings.settings.shortcutsSection.title), description: t(strings.settings.shortcutsSection.description) }), _jsxs(SettingsCard, { className: "space-y-4", children: [_jsxs("div", { className: "flex items-center justify-between gap-3", children: [_jsxs("div", { children: [_jsx("div", { className: "text-sm font-medium text-wc-text-secondary", children: t(strings.settings.shortcutsSection.profilesTitle) }), _jsx("div", { className: "text-[11px] text-wc-text-muted", children: t(strings.settings.shortcutsSection.profilesHint) })] }), _jsxs(Button, { "data-testid": "create-profile", variant: "outline", size: "sm", className: "h-8 px-3 text-xs", onClick: handleCreateProfile, children: [_jsx(Plus, { className: "me-1 h-3 w-3" }), t(strings.settings.shortcutsSection.newProfile)] })] }), profileError && (_jsxs("div", { "data-testid": "settings-error", className: "flex items-start gap-2 rounded-xl border border-wc-error bg-wc-error-surface px-3 py-2 text-xs text-wc-error-detail", children: [_jsx(AlertCircle, { className: "mt-0.5 h-3.5 w-3.5 shrink-0" }), _jsx("span", { children: profileError })] })), profileLoading ? (_jsx("div", { className: "py-4 text-center text-xs text-wc-text-faint", children: t(strings.settings.shortcutsSection.loading) })) : profiles.length === 0 ? (_jsx("div", { className: "py-4 text-center text-xs text-wc-text-faint", children: t(strings.settings.shortcutsSection.empty) })) : (_jsx("div", { className: "space-y-3", children: profiles.map((profile) => (_jsx(ShortcutEditor, { profile: profile, onSave: handleSaveProfile, onDelete: handleDeleteProfile }, profile.id))) }))] })] }));
}
