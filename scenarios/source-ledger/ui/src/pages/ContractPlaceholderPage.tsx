export interface ContractPlaceholderPageProps {
  title: string;
  description: string;
}

export function ContractPlaceholderPage({ title, description }: ContractPlaceholderPageProps) {
  return (
    <section className="flex flex-col gap-3" aria-labelledby="contract-placeholder-heading">
      <h2 id="contract-placeholder-heading" className="text-2xl font-semibold">{title}</h2>
      <p className="text-app-muted-foreground">{description}</p>
    </section>
  );
}
