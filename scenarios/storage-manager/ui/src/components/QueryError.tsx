export function QueryError({ message, testId }: { message: string; testId: string }) {
  return (
    <div data-testid={testId} className="rounded-panel border border-app-danger/40 bg-app-danger/10 p-4 text-sm">
      <p className="font-semibold text-app-danger">Unable to load this surface</p>
      <p className="mt-1 text-app-muted-foreground">{message}</p>
    </div>
  );
}
