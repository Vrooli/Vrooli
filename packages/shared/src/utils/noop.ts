/** 
 * Placeholder to satisfy Formik's `onSubmit` prop, if you 
 * are submitting the form elsewhere.
 **/
export const noopSubmit = (values: unknown) => {
    console.warn("Formik onSubmit called unexpectedly with values:", values);
};

export const noop = () => {
    console.warn("Noop called");
};
