/**
 * Updates --card-background-color, --card-primary-color, and --card-secondary-color
 */
function updateCardBorderColor() {
    var root = document.documentElement;
    var body = document.getElementsByTagName('body')[0];
    var isDarkMode = body.getAttribute('data-md-color-scheme') === 'slate';

    if (isDarkMode) {
        root.style.setProperty("--card-background-color", "#607d8b"); // Blue-grey color
        root.style.setProperty("--card-primary-color", "#fff");
        root.style.setProperty("--card-secondary-color", "#fff");
    } else {
        root.style.setProperty("--card-background-color", "#3f51b5"); // Indigo color
        root.style.setProperty("--card-primary-color", "#fff");
        root.style.setProperty("--card-secondary-color", "#fff");
    }
}

document.addEventListener("DOMContentLoaded", function () {
    updateCardBorderColor();

    // Watch for changes in the data-md-color-scheme attribute
    var body = document.getElementsByTagName('body')[0];
    var observer = new MutationObserver(function (mutations) {
        mutations.forEach(function (mutation) {
            if (mutation.attributeName === 'data-md-color-scheme') {
                updateCardBorderColor();
            }
        });
    });

    observer.observe(body, {
        attributes: true,
        attributeFilter: ['data-md-color-scheme']
    });
});