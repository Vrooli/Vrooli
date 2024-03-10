import { FormControl, FormControlProps, FormHelperText, InputAdornment, InputLabel, List, ListItem, OutlinedInput, Popover, TextField, Typography, useTheme } from '@mui/material';
import { useField } from 'formik';
import { PhoneIcon } from 'icons';
import { CountryCallingCode, CountryCode } from 'libphonenumber-js';
import { useEffect, useMemo, useState } from 'react';
import { PhoneNumberInputProps } from '../types';

export const PhoneNumberInput = ({
    autoComplete = "tel",
    autoFocus = false,
    fullWidth = true,
    label,
    name,
    ...props
}: PhoneNumberInputProps) => {
    const { palette } = useTheme();
    const [field, meta, helpers] = useField<string>(name);
    const [selectedCountry, setSelectedCountry] = useState<CountryCode>("US");
    const [phoneNumber, setPhoneNumber] = useState(typeof field.value === "string" ? field.value : "");
    const [filter, setFilter] = useState('');
    const [isValid, setIsValid] = useState(true);

    // Country select popover
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const openPopover = (event: React.MouseEvent<HTMLElement>) => { setAnchorEl(event.currentTarget); };
    const closePopover = () => { setAnchorEl(null); };

    // Validate phone number
    useEffect(() => {
        const libphonenumber = import('libphonenumber-js');
        libphonenumber.then((lib) => {
            try {
                // Make sure field and country are valid types
                if (typeof field.value !== "string" || selectedCountry === null) {
                    setIsValid(false);
                    helpers.setError("Invalid phone number");
                    return;
                }
                // Parse phone number
                const parsedNumber = lib.parsePhoneNumber(field.value);
                // Make sure parsed country matches selected country
                if (parsedNumber?.country !== selectedCountry) {
                    setIsValid(false);
                    helpers.setError("Invalid phone number");
                    return;
                }
                console.log('made it past country validation check', parsedNumber, selectedCountry, field.value)
                // Perform normal validation
                const valid = lib.isValidPhoneNumber(field.value, selectedCountry);
                if (valid) {
                    setIsValid(true);
                    helpers.setError(undefined);
                    helpers.setValue(parsedNumber.number);
                } else {
                    setIsValid(false);
                    helpers.setError("Invalid phone number");
                }
            } catch (error) {
                console.error(error);
                setIsValid(false);
            }
        });
    }, [field.value, selectedCountry]);


    const handleCountryChange = async (newCountry: CountryCode) => {
        const libphonenumber = await import('libphonenumber-js');
        setSelectedCountry(newCountry);
        helpers.setValue(`+${libphonenumber.getCountryCallingCode(newCountry)}${phoneNumber.replace(/[^0-9]/g, '')}`);
        closePopover();
    };

    const handlePhoneNumberChange = (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        const newPhoneNumber = event.target.value;
        const cursorPosition = event.target.selectionStart ?? 0;
        const isBackspaceOnFormattingChar = phoneNumber[cursorPosition] === ')' && newPhoneNumber.length < phoneNumber.length;

        const libphonenumber = import('libphonenumber-js');
        libphonenumber.then(async (lib) => {
            if (isBackspaceOnFormattingChar) {
                // Adjust the phone number by removing the character before the formatting character
                const adjustedPhoneNumber = phoneNumber.slice(0, cursorPosition - 1) + phoneNumber.slice(cursorPosition);
                setPhoneNumber(adjustedPhoneNumber);
                helpers.setValue(`+${lib.getCountryCallingCode(selectedCountry ?? "US")}${adjustedPhoneNumber.replace(/[^0-9]/g, '')}`);
                // Optionally, set the cursor position to ensure it remains in the correct place
                setTimeout(() => event.target.setSelectionRange(cursorPosition - 1, cursorPosition - 1), 0);
            } else {
                // Proceed with normal handling if not dealing with a special case
                // Get AsYouType instance for phone number formatting
                const asYouType = new lib.AsYouType(selectedCountry);
                const formattedNumber = asYouType.input(newPhoneNumber);
                // Use it to set the shown phone number
                setPhoneNumber(formattedNumber);
                // If the phone number is empty, set field.value to an empty string
                if (formattedNumber.trim() === '') {
                    helpers.setValue('');
                } else {
                    // If the number does not include a country code, add it when setting field value
                    let parsedNumber: { country?: CountryCode | undefined } = {};
                    try {
                        parsedNumber = lib.parsePhoneNumber(newPhoneNumber);
                    } catch (error) {
                        console.error(error);
                    }
                    console.log('got parsed number for country check', parsedNumber, newPhoneNumber);
                    const withCountry = parsedNumber.country ? formattedNumber : `+${lib.getCountryCallingCode(selectedCountry)}${formattedNumber}`;
                    helpers.setValue(withCountry);
                }
            }
        });
    };

    const [filteredCountries, setFilteredCountries] = useState<{ country: CountryCode, num: CountryCallingCode }[]>([]);
    useMemo(() => {
        const filterCountries = async () => {
            const libphonenumber = await import('libphonenumber-js');
            return libphonenumber.getCountries().filter((country) => (
                country.toLowerCase().includes(filter.toLowerCase()) ||
                libphonenumber.getCountryCallingCode(country as CountryCode).includes(filter)
            )).map((country) => ({ country, num: libphonenumber.getCountryCallingCode(country as CountryCode) }));
        };
        filterCountries().then(setFilteredCountries);
    }, [filter]);

    return (
        <>
            <FormControl fullWidth={fullWidth} variant="outlined" error={!isValid || (meta.touched && !!meta.error)} {...props as FormControlProps}>
                <InputLabel htmlFor={name}>{label ?? "Phone Number"}</InputLabel>
                <OutlinedInput
                    id={name}
                    name={name}
                    type="tel"
                    value={phoneNumber}
                    onChange={handlePhoneNumberChange}
                    autoFocus={autoFocus}
                    startAdornment={
                        <InputAdornment position="start" onClick={openPopover} sx={{ cursor: "pointer" }}>
                            <PhoneIcon style={{ marginRight: "4px" }} />
                            <Typography variant="body1" sx={{ marginRight: "4px" }}>{selectedCountry}</Typography>
                        </InputAdornment>
                    }
                    error={meta.touched && !!meta.error}
                    label={label ?? "Phone Number"}
                    sx={{
                        '& .MuiOutlinedInput-notchedOutline': {
                            borderColor: !isValid || (meta.touched && !!meta.error) ? palette.error.main : '',
                        },
                        '&:hover .MuiOutlinedInput-notchedOutline': {
                            borderColor: !isValid || (meta.touched && !!meta.error) ? palette.error.main : '',
                        },
                        '&.Mui-focused .MuiOutlinedInput-notchedOutline': {
                            borderColor: !isValid || (meta.touched && !!meta.error) ? palette.error.main : palette.primary.main, // Keep primary color on focus if not invalid
                        },
                    }}
                />
            </FormControl>
            {(!isValid || meta.touched) && meta.error && (
                <FormHelperText error>{meta.error}</FormHelperText>
            )}
            <Popover
                open={Boolean(anchorEl)}
                anchorEl={anchorEl}
                onClose={closePopover}
                anchorOrigin={{
                    vertical: 'bottom',
                    horizontal: 'left',
                }}
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
                <List style={{ maxHeight: 300, overflow: 'auto' }}>
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
};
