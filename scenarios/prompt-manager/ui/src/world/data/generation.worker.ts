import { createGenerationService, type GenerationRequest, type GenerationReply } from './generation'

// Keep the worker boundary free of Window and rendering-library objects.
const scope = self as unknown as {
  postMessage(reply: GenerationReply, transfers: ArrayBuffer[]): void
  onmessage: ((event: MessageEvent<GenerationRequest>) => void) | null
}
const service = createGenerationService((reply, transfers = []) => scope.postMessage(reply, transfers))
scope.onmessage = event => { void service.receive(event.data) }
