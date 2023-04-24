import { renderApp } from ":local/ui";
import { createRef } from "react";
import { renderToString } from "react-dom/server";

export default function render(req) {
    const element = createRef<any>();
    renderApp(element);

    const html = renderToString(element.current);
    return html;
}
