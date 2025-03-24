import { register } from "node:module";
import { pathToFileURL } from "node:url";

register(new URL("./loader.js", import.meta.url).href, pathToFileURL("./").href);
