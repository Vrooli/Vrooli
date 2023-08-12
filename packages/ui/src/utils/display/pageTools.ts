import { ViewDisplayType } from "views/types";

export const toDisplay = (isOpen?: boolean) => isOpen !== undefined ? "dialog" : "page" as ViewDisplayType;
