export function shouldOpenBacklogRow(target: EventTarget | null): boolean {
  return !(target instanceof HTMLElement && Boolean(target.closest("button,a,input,textarea,select,[role='button'],[role='link']")));
}
