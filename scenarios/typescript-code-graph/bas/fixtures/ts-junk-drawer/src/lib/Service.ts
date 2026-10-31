export class Service {
  private name: string;
  constructor(name: string) {
    this.name = name;
  }
  describe(): string {
    return `service:${this.name}`;
  }
}
