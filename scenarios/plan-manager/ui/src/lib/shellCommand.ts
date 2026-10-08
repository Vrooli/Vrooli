export function shellQuote(arg: string) {
  if (/^[A-Za-z0-9_./:=@+-]+$/.test(arg)) return arg;
  return `'${arg.replace(/'/g, "'\\''")}'`;
}

export function shellCommand(argv: readonly string[], prefix: readonly string[] = []) {
  return [...prefix, ...argv].map(shellQuote).join(" ");
}
