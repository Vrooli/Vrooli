import Box from "@mui/material/Box";
import FormControl from "@mui/material/FormControl";
import FormHelperText from "@mui/material/FormHelperText";
import InputAdornment from "@mui/material/InputAdornment";
import InputLabel from "@mui/material/InputLabel";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import OutlinedInput from "@mui/material/OutlinedInput";
import Popover from "@mui/material/Popover";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import type { FormControlProps } from "@mui/material";
import { useTheme } from "@mui/material";
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
                
                // Handle empty value case
                if (!value || value.trim() === "") {
                    stableSetError(undefined);
                    return;
                }
                
                // Parse phone number with error handling
                let parsedNumber;
                try {
                    parsedNumber = lib.parsePhoneNumber(value);
                } catch (parseError) {
                    // If parsing fails, set error but don't crash
                    stableSetError("Invalid phone number");
                    return;
                }
                
                // Check if parsed number exists and has valid country
                if (!parsedNumber) {
                    stableSetError("Invalid phone number");
                    return;
                }
                
                // Make sure parsed country matches selected country
                if (parsedNumber.country !== selectedCountry) {
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
            try {
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
                        // Silently handle parse errors for partial numbers
                        console.debug("Phone number parsing failed for partial input:", error);
                    }
                    const withCountry = parsedNumber.country ? formattedNumber : `+${lib.getCountryCallingCode(selectedCountry)}${formattedNumber}`;
                    stableOnChange(withCountry);
                }
            } catch (error) {
                console.error("Phone number formatting failed:", error);
                // Fallback to basic formatting
                setPhoneNumber(phoneNumberRef.current);
                stableOnChange(phoneNumberRef.current);
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
                        autoComplete={autoComplete}
                        autoFocus={autoFocus}
                        data-testid="phone-number-input"
                        startAdornment={
                            <InputAdornment position="start" onClick={openPopover} sx={{ cursor: "pointer" }} data-testid="country-selector-button">
                                <IconCommon
                                    decorative
                                    name="Phone"
                                    style={{ marginRight: "4px" }}
                                />
                                <Typography variant="body1" sx={{ marginRight: "4px" }} data-testid="selected-country">{selectedCountry}</Typography>
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
                data-testid="country-selector-popover"
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
                    data-testid="country-search-input"
                />
                <List style={{ maxHeight: 300, overflow: "auto" }} data-testid="country-list">
                    {filteredCountries.map(({ country, num }) => (
                        <ListItem
                            button
                            key={country}
                            onClick={() => handleCountryChange(country)}
                            data-testid={`country-option-${country}`}
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
