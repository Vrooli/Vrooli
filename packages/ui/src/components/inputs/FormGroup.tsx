import React, { forwardRef } from "react";
import { FormGroupProps } from "./types.js";
import { cn } from "../../utils/tailwind-theme.js";
import { getFormGroupStyles } from "./formGroupStyles.js";

export const FormGroup = forwardRef<HTMLDivElement, FormGroupProps>(({
    children,
    row = false,
    className,
    style,
    sx,
    ...props
}, ref) => {
    const styles = getFormGroupStyles({
        row,
        sx,
    });

    return (
        <div
            ref={ref}
            className={cn(styles.root, className)}
            style={style}
            role="group"
            {...props}
        >
            {children}
        </div>
    );
});

FormGroup.displayName = "FormGroup";