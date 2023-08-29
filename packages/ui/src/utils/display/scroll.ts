export const scrollIntoFocusedView = (id: string) => {
    const element = document.getElementById(id);
    if (element) {
        element.scrollIntoView({ behavior: "smooth" });
        // Delay the focus slightly to allow smooth scrolling to complete
        setTimeout(() => {
            element.focus();
        }, 300);
    } else {
        console.warn(`Could not find element with id ${id}`);
    }
};
