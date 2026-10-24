import type { Page } from 'rebrowser-playwright';
import type { SessionState } from '../types';

/** Maps driver page identities to Playwright handles without owning tab order. */
export class DriverPageBindings {
  private readonly byId = new Map<string, Page>();
  private readonly idByPage = new WeakMap<Page, string>();
  private readonly attached = new WeakSet<Page>();

  attachContext(session: SessionState): void {
    session.context.on('page', (page) => this.attachPage(session, page));
    for (const page of session.context.pages()) this.attachPage(session, page);
  }

  private attachPage(session: SessionState, page: Page): void {
    if (this.attached.has(page)) return;
    this.attached.add(page);
    page.once('close', () => {
      this.remove(page);
      if (session.page === page) {
        session.frameStack.length = 0;
        const next = session.context.pages().find((candidate) => !candidate.isClosed());
        if (next) session.page = next;
      }
    });
  }

  register(page: Page, id: string): string {
    const existing = this.idByPage.get(page);
    if (existing) return existing;
    this.byId.set(id, page);
    this.idByPage.set(page, id);
    return id;
  }

  getPage(id: string): Page | undefined { return this.byId.get(id); }
  getId(page: Page): string | undefined { return this.idByPage.get(page); }
  has(page: Page): boolean { return this.idByPage.has(page); }
  ids(): string[] { return [...this.byId.keys()]; }

  remove(page: Page): void {
    const id = this.idByPage.get(page);
    if (id) this.byId.delete(id);
    this.idByPage.delete(page);
  }

  retain(page: Page): void {
    for (const [id, tracked] of this.byId) {
      if (tracked === page) continue;
      this.byId.delete(id);
      this.idByPage.delete(tracked);
    }
  }
}
