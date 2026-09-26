export interface Request {
  id: string;
}

export interface Response {
  ok: boolean;
}

export function createConnectClient() {
  return {
    submit(input: Request): Response {
      return { ok: input.id.length > 0 };
    },
  };
}
