/** 
 * Placeholder to satisfy Formik's `onSubmit` prop, if you 
 * are submitting the form elsewhere.
 **/
export function noopSubmit(values: unknown) {
    console.warn("Formik onSubmit called unexpectedly with values:", values);
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