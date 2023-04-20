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

    var themeToggle = document.querySelector(".md-palette-toggle");
    if (themeToggle) {
        themeToggle.addEventListener("click", function () {
            setTimeout(updateCardBorderColor, 100);
        });
    }
});

if (window.matchMedia) {
    var mediaQueryList = window.matchMedia('print');
    mediaQueryList.addListener(function (mql) {
        if (mql.matches) {
            updateCardBorderColor();
        } else {
            updateCardBorderColor();
        }
    });
}

window.onbeforeprint = updateCardBorderColor;
window.onafterprint = updateCardBorderColor;