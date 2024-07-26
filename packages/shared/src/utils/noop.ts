/** 
 * Placeholder to satisfy Formik's `onSubmit` prop, if you 
 * are submitting the form elsewhere.
 **/
export function noopSubmit(values: unknown, helpers: { setSubmitting: ((submitting: boolean) => unknown) }) {
    console.warn("Formik onSubmit called unexpectedly with values:", values);
    helpers.setSubmitting(false);
}

export function noop() {
    console.warn("Noop called");
}

export function funcTrue() {
    return true;
}

export function funcFalse() {
    return false;
}

export function preventFormSubmit(event: { preventDefault: (() => unknown), stopPropagation: (() => unknown) }) {
    console.log("in preventFormSubmit");
    event.preventDefault();
    event.stopPropagation();
}
