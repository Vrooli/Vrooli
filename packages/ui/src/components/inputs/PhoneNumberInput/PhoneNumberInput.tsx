import { Box, FormControl, type FormControlProps, FormHelperText, InputAdornment, InputLabel, List, ListItem, OutlinedInput, Popover, TextField, Typography, useTheme } from "@mui/material";
import { useField } from "formik";
import { type CountryCallingCode, type CountryCode } from "libphonenumber-js";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useDebounce } from "../../../hooks/useDebounce.js";
import { usePopover } from "../../../hooks/usePopover.js";
import { useStableCallback } from "../../../hooks/useStableCallback.js";
import { IconCommon } from "../../../icons/Icons.js";
import { type PhoneNumberInputBaseProps, type PhoneNumberInputProps } from "../types.js";

const anchorOrigin = {
    vertical: "bottom",
    horizontal: "left",
} as const;

export function PhoneNumberInputBase({
    autoComplete = "tel",
    autoFocus = false,
    error,
    fullWidth = true,
    helperText,
    label,
    name,
    onChange,
    setError,
    value,
    ...props
}: PhoneNumberInputBaseProps) {
    const { palette } = useTheme();
    const [selectedCountry, setSelectedCountry] = useState<CountryCode>("US");
    const [phoneNumber, setPhoneNumber] = useState(typeof value === "string" ? value : "");
    const phoneNumberRef = useRef(phoneNumber);
    const [filter, setFilter] = useState("");
    const stableOnChange = useStableCallback(onChange);
    const stableSetError = useStableCallback(setError);

    // Country select popover
    const [anchorEl, openPopover, closePopover] = usePopover();

    // Validate phone number
    useEffect(function validatePhoneNumberEffect() {
        const libphonenumber = import("libphonenumber-js");
        libphonenumber.then((lib) => {
            try {
                // Make sure field and country are valid types
                if (typeof value !== "string" || selectedCountry === null) {
                    stableSetError("Invalid phone number");
                    return;
                }
                // Parse phone number
                const parsedNumber = lib.parsePhoneNumber(value);
                // Make sure parsed country matches selected country
                if (parsedNumber?.country !== selectedCountry) {
                    stableSetError("Invalid phone number");
                    return;
                }
                // Perform normal validation
                const valid = lib.isValidPhoneNumber(value, selectedCountry);
                if (valid) {
                    stableSetError(undefined);
                    if (value !== parsedNumber.number) {
                        stableOnChange(parsedNumber.number);
                    }
                } else {
                    stableSetError("Invalid phone number");
                }
            } catch (error) {
                console.error(error);
                stableSetError("Invalid phone number");
            }
        });
    }, [selectedCountry, stableOnChange, stableSetError, value]);

    const handleCountryChange = useCallback(async (newCountry: CountryCode) => {
        const libphonenumber = await import("libphonenumber-js");
        setSelectedCountry(newCountry);
        stableOnChange(`+${libphonenumber.getCountryCallingCode(newCountry)}${phoneNumberRef.current.replace(/[^0-9]/g, "")}`);
        closePopover();
    }, [stableOnChange]);

    const handlePhoneNumberChange = useCallback(() => {
        const libphonenumber = import("libphonenumber-js");
        libphonenumber.then(async (lib) => {
            // Get AsYouType instance for phone number formatting
            const asYouType = new lib.AsYouType(selectedCountry);
            const formattedNumber = asYouType.input(phoneNumberRef.current);
            // Use it to set the shown phone number
            setPhoneNumber(formattedNumber);
            phoneNumberRef.current = formattedNumber;
            // If the phone number is empty, set value to an empty string
            if (formattedNumber.trim() === "") {
                stableOnChange("");
            } else {
                // If the number does not include a country code, add it when setting field value
                let parsedNumber: { country?: CountryCode | undefined } = {};
                try {
                    parsedNumber = lib.parsePhoneNumber(phoneNumberRef.current);
                } catch (error) {
                    console.error(error);
                }
                const withCountry = parsedNumber.country ? formattedNumber : `+${lib.getCountryCallingCode(selectedCountry)}${formattedNumber}`;
                stableOnChange(withCountry);
            }
        });
    }, [stableOnChange, selectedCountry]);
    const [handlePhoneNumberChangeDebounced] = useDebounce(handlePhoneNumberChange, 50);

    const handleImmediateChange = useCallback((event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        let value = event.target.value;
        const cursorPosition = event.target.selectionStart ?? 0;
        // If you backspaced on a formatting character, remove an additional character 
        // (so that you remove the next number, and it doesn't just reformat the same number)
        const isBackspaceOnFormattingChar = phoneNumberRef.current[cursorPosition] === ")" && value.length < phoneNumberRef.current.length;
        if (isBackspaceOnFormattingChar) {
            value = value.slice(0, cursorPosition - 1) + value.slice(cursorPosition);
        }
        // Change the phone number to the new value
        setPhoneNumber(value);
        phoneNumberRef.current = value;
        // Trigger debounce for formatting and validation
        handlePhoneNumberChangeDebounced(event);
    }, [handlePhoneNumberChangeDebounced]);

    const [filteredCountries, setFilteredCountries] = useState<{ country: CountryCode, num: CountryCallingCode }[]>([]);
    useMemo(() => {
        async function filterCountries() {
            const libphonenumber = await import("libphonenumber-js");
            return libphonenumber.getCountries().filter((country) => (
                country.toLowerCase().includes(filter.toLowerCase()) ||
                libphonenumber.getCountryCallingCode(country as CountryCode).includes(filter)
            )).map((country) => ({ country, num: libphonenumber.getCountryCallingCode(country as CountryCode) }));
        }
        filterCountries().then(setFilteredCountries);
    }, [filter]);

    return (
        <>
            <Box width="-webkit-fill-available">
                <FormControl fullWidth={fullWidth} variant="outlined" error={!!error} {...props as FormControlProps}>
                    <InputLabel htmlFor={name}>{label ?? "Phone Number"}</InputLabel>
                    <OutlinedInput
                        id={name}
                        name={name}
                        type="tel"
                        value={phoneNumber}
                        onChange={handleImmediateChange}
                        autoFocus={autoFocus}
                        startAdornment={
                            <InputAdornment position="start" onClick={openPopover} sx={{ cursor: "pointer" }}>
                                <IconCommon
                                    decorative
                                    name="Phone"
                                    style={{ marginRight: "4px" }}
                                />
                                <Typography variant="body1" sx={{ marginRight: "4px" }}>{selectedCountry}</Typography>
                            </InputAdornment>
                        }
                        error={!!error}
                        label={label ?? "Phone Number"}
                        sx={{
                            "& .MuiOutlinedInput-notchedOutline": {
                                borderColor: error ? palette.error.main : "",
                            },
                            "&:hover .MuiOutlinedInput-notchedOutline": {
                                borderColor: error ? palette.error.main : "",
                            },
                            "&.Mui-focused .MuiOutlinedInput-notchedOutline": {
                                borderColor: error ? palette.error.main : palette.primary.main, // Keep primary color on focus if not invalid
                            },
                        }}
                    />
                </FormControl>
                {helperText && <FormHelperText id={`helper-text-${name}`}>{typeof helperText === "string" ? helperText : JSON.stringify(helperText)}</FormHelperText>}
            </Box>
            <Popover
                open={Boolean(anchorEl)}
                anchorEl={anchorEl}
                onClose={closePopover}
                anchorOrigin={anchorOrigin}
            >
                <TextField
                    margin="normal"
                    variant="outlined"
                    fullWidth
                    label="Search country or code"
                    autoComplete="off"
                    autoFocus
                    name="countrySearchNoFill" // Weird name to discourage autofill
                    value={filter}
                    onChange={(e) => setFilter(e.target.value)}
                />
                <List style={{ maxHeight: 300, overflow: "auto" }}>
                    {filteredCountries.map(({ country, num }) => (
                        <ListItem
                            button
                            key={country}
                            onClick={() => handleCountryChange(country)}
                        >
                            {country} (+{num})
                        </ListItem>
                    ))}
                </List>
            </Popover>
        </>
    );
}

export function PhoneNumberInput({
    name,
    onChange,
    ...props
}: PhoneNumberInputProps) {
    const [field, meta, helpers] = useField(name);

    function handleChange(newValue: string) {
        if (onChange) onChange(newValue);
        helpers.setValue(newValue);
    }

    function handleSetError(error: string | undefined) {
        helpers.setError(error);
    }

    return (
        <PhoneNumberInputBase
            {...props}
            name={name}
            value={field.value}
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error}
            onBlur={field.onBlur}
            onChange={handleChange}
            setError={handleSetError}
        />
    );
}
