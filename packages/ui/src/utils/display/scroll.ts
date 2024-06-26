/** A delay to wait for smooth scrolling to complete before focusing */
const SCROLL_FOCUS_DELAY = 300;

export function scrollIntoFocusedView(id: string) {
    const element = document.getElementById(id);
    if (element) {
        element.scrollIntoView({ behavior: "smooth" });
        setTimeout(() => { element.focus(); }, SCROLL_FOCUS_DELAY);
    } else {
        console.warn(`Could not find element with id ${id}`);
    }
}
