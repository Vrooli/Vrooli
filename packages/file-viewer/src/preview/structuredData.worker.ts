import { parseStructure } from "./parseStructure.js";

self.onmessage = (event: MessageEvent<{ source: string; json: boolean }>) => {
  self.postMessage(parseStructure(event.data.source, event.data.json));
};
