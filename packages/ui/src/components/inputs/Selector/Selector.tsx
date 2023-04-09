import { useField } from 'formik';
import { SelectorBase } from '../SelectorBase/SelectorBase';
import { SelectorProps } from '../types';

export const Selector = <T extends string | number | { [x: string]: any }>({
    name,
    onChange,
    ...props
}: SelectorProps<T>) => {
    const [field, meta] = useField(name);

    return (
        <SelectorBase
            {...props}
            name={name}
            value={field.value}
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error}
            onBlur={field.onBlur}
            onChange={(newValue) => {
                if (onChange) onChange(newValue);
                field.onChange(newValue);
            }}
        />
    );
};