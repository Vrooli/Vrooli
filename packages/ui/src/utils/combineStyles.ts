import { Theme } from "@material-ui/core";

export const combineStyles: any = (...styles: Array<Theme | {} | string>) => {
    return function CombineStyles(theme: Theme) {
        const outStyles = styles.map((arg: Theme | {} | string) => {
            // Apply the "theme" object for style functions.
            if (typeof arg === 'function') {
                return arg(theme);
            }
            // Objects need no change.
            return arg;
        });
        return outStyles.reduce((acc, val) => Object.assign(acc, val));
    };
}