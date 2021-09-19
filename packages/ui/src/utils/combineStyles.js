export function combineStyles(...styles) {
    return function CombineStyles(theme) {
        const outStyles = styles.map((arg) => {
            // Apply the "theme" object for style functions.
            if (typeof arg === 'function') {
                return arg(theme);
            }
            // Objects need no change.
            return arg;
        });

        console.log('COMBINING STYLES', outStyles)
        return outStyles.reduce((acc, val) => Object.assign(acc, val));
    };
}