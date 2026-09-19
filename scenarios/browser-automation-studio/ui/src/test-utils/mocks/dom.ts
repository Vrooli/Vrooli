type WindowWithDragEvent = Window & { DragEvent?: typeof DragEvent };

function createEmptyDataTransfer(): DataTransfer {
  return {
    dropEffect: 'move',
    effectAllowed: 'all',
    files: [] as unknown as FileList,
    items: [] as unknown as DataTransferItemList,
    types: [],
    setData: () => {},
    getData: () => '',
    clearData: () => {},
    setDragImage: () => {},
  } as DataTransfer;
}

export function installDragEventShim(targetWindow: Window = window) {
  const windowWithDragEvent = targetWindow as WindowWithDragEvent;

  if (typeof windowWithDragEvent.DragEvent !== 'undefined') {
    return;
  }

  class DragEventPolyfill extends Event {
    dataTransfer: DataTransfer;

    constructor(
      type: string,
      eventInitDict?: DragEventInit & { dataTransfer?: DataTransfer }
    ) {
      super(type, eventInitDict);
      this.dataTransfer = eventInitDict?.dataTransfer ?? createEmptyDataTransfer();
    }
  }

  windowWithDragEvent.DragEvent = DragEventPolyfill as typeof DragEvent;
}
