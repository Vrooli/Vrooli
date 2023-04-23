import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { BumpMajorIcon, BumpMinorIcon, BumpModerateIcon } from "@local/icons";
import { calculateVersionsFromString, meetsMinVersion } from "@local/validation";
import { Stack, TextField, Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useMemo, useRef, useState } from "react";
import { getMinimumVersion } from "../../../utils/shape/general";
import { ColorIconButton } from "../../buttons/ColorIconButton/ColorIconButton";
export const VersionInput = ({ autoFocus = false, fullWidth = true, label = "Version", name = "versionLabel", versions, ...props }) => {
    const { palette } = useTheme();
    const textFieldRef = useRef(null);
    const [field, meta, helpers] = useField(name);
    const [internalValue, setInternalValue] = useState(field.value);
    const handleChange = useCallback((event) => {
        const newValue = event.target.value;
        setInternalValue(newValue);
        if (newValue.match(/^[0-9]+(\.[0-9]+){0,2}$/)) {
            if (meetsMinVersion(newValue, getMinimumVersion(versions))) {
                helpers.setValue(newValue);
            }
        }
    }, [helpers, versions]);
    const { major, moderate, minor } = useMemo(() => calculateVersionsFromString(internalValue ?? ""), [internalValue]);
    const bumpMajor = useCallback(() => {
        const changedVersion = `${major + 1}.${moderate}.${minor}`;
        helpers.setValue(changedVersion);
    }, [major, moderate, minor, helpers]);
    const bumpModerate = useCallback(() => {
        const changedVersion = `${major}.${moderate + 1}.${minor}`;
        helpers.setValue(changedVersion);
    }, [major, moderate, minor, helpers]);
    const bumpMinor = useCallback(() => {
        const changedVersion = `${major}.${moderate}.${minor + 1}`;
        helpers.setValue(changedVersion);
    }, [major, moderate, minor, helpers]);
    const handleBlur = useCallback((ev) => {
        const changedVersion = `${major}.${moderate}.${minor}`;
        helpers.setValue(changedVersion);
        field.onBlur(ev);
    }, [major, moderate, minor, helpers, field]);
    return (_jsxs(Stack, { direction: "row", spacing: 0, children: [_jsx(TextField, { autoFocus: autoFocus, fullWidth: true, id: "versionLabel", name: "versionLabel", label: label, value: internalValue, onBlur: handleBlur, onChange: handleChange, error: meta.touched && !!meta.error, helperText: meta.touched && meta.error, ref: textFieldRef, sx: {
                    "& .MuiInputBase-root": {
                        borderRadius: "5px 0 0 5px",
                    },
                } }), _jsx(Tooltip, { placement: "top", title: "Major bump (increment the first number)", children: _jsx(ColorIconButton, { "aria-label": 'major-bump', background: palette.secondary.main, onClick: bumpMajor, sx: {
                        borderRadius: "0",
                        borderRight: `1px solid ${palette.secondary.contrastText}`,
                        height: `${textFieldRef.current?.clientHeight ?? 56}px)`,
                    }, children: _jsx(BumpMajorIcon, {}) }) }), _jsx(Tooltip, { placement: "top", title: "Moderate bump (increment the middle number)", children: _jsx(ColorIconButton, { "aria-label": 'moderate-bump', background: palette.secondary.main, onClick: bumpModerate, sx: {
                        borderRadius: "0",
                        borderRight: `1px solid ${palette.secondary.contrastText}`,
                        height: `${textFieldRef.current?.clientHeight ?? 56}px)`,
                    }, children: _jsx(BumpModerateIcon, {}) }) }), _jsx(Tooltip, { placement: "top", title: "Minor bump (increment the last number)", children: _jsx(ColorIconButton, { "aria-label": 'minor-bump', background: palette.secondary.main, onClick: bumpMinor, sx: {
                        borderRadius: "0 5px 5px 0",
                        height: `${textFieldRef.current?.clientHeight ?? 56}px)`,
                    }, children: _jsx(BumpMinorIcon, {}) }) })] }));
};
//# sourceMappingURL=VersionInput.js.map