import { parseStructure } from "./parseStructure";

self.onmessage = (event: MessageEvent<{ source: string; json: boolean }>) => {
  self.postMessage(parseStructure(event.data.source, event.data.json));
};
