import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { categoriesClient, type Category } from "../../api/categories";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { Input } from "../../components/ui/input";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

const categoriesKey = ["categories"] as const;

function CategoryRow({ category }: { category: Category }) {
  const { t } = useTranslation();
  const client = useQueryClient();
  const [name, setName] = useState(category.name);
  const [description, setDescription] = useState(category.description);
  const rename = useMutation({
    mutationFn: () => categoriesClient.renameCategory({ id: category.id, name: name.trim(), description: description.trim() }),
    onSuccess: () => void client.invalidateQueries({ queryKey: categoriesKey }),
  });
  const retire = useMutation({
    mutationFn: () => categoriesClient.retireCategory({ id: category.id }),
    onSuccess: () => void client.invalidateQueries({ queryKey: categoriesKey }),
  });

  return (
    <li className="flex flex-col gap-2 rounded border border-app-border p-3">
      <div className="flex items-center justify-between gap-3">
        <strong>{category.reserved ? `${category.name} (${t(strings.categories.reserved)})` : category.name}</strong>
        {category.reserved ? null : <Button variant="danger" size="sm" onClick={() => retire.mutate()} disabled={retire.isPending}>{t(strings.categories.retire)}</Button>}
      </div>
      {category.reserved ? <p className="text-sm text-app-muted-foreground">{t(strings.categories.fallback)}</p> : <>
        <Input aria-label={t(strings.categories.nameFor, { name: category.name })} value={name} onChange={(event) => setName(event.target.value)} />
        <Input aria-label={t(strings.categories.descriptionFor, { name: category.name })} value={description} onChange={(event) => setDescription(event.target.value)} placeholder={t(strings.categories.descriptionInput)} />
        <div><Button variant="secondary" size="sm" onClick={() => rename.mutate()} disabled={!name.trim() || rename.isPending}>{t(strings.categories.save)}</Button></div>
        {(rename.error || retire.error) && <p className="text-sm text-app-danger">{t(strings.categories.changeError)}</p>}
      </>}
    </li>
  );
}

// Categories are runtime data, deliberately not seed literals in the product.
// The reserved fallback is created by the API and cannot be renamed or retired.
export function CategoriesCard() {
  const { t } = useTranslation();
  const client = useQueryClient();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const list = useQuery({ queryKey: categoriesKey, queryFn: () => categoriesClient.listCategories({}) });
  const create = useMutation({
    mutationFn: () => categoriesClient.createCategory({ name: name.trim(), description: description.trim() }),
    onSuccess: () => {
      setName("");
      setDescription("");
      void client.invalidateQueries({ queryKey: categoriesKey });
    },
  });

  return (
    <Card aria-label={t(strings.categories.title)}>
      <CardHeader><CardTitle>{t(strings.categories.title)}</CardTitle></CardHeader>
      <CardContent className="flex flex-col gap-3">
        <p className="text-sm text-app-muted-foreground">{t(strings.categories.description)}</p>
        <div className="grid gap-2 sm:grid-cols-2">
          <Input aria-label={t(strings.categories.newName)} value={name} onChange={(event) => setName(event.target.value)} placeholder={t(strings.categories.name)} />
          <Input aria-label={t(strings.categories.newDescription)} value={description} onChange={(event) => setDescription(event.target.value)} placeholder={t(strings.categories.descriptionInput)} />
        </div>
        <div><Button onClick={() => create.mutate()} disabled={!name.trim() || create.isPending}>{create.isPending ? t(strings.categories.creating) : t(strings.categories.create)}</Button></div>
        {create.error && <p className="text-sm text-app-danger">{t(strings.categories.createError)}</p>}
        {list.isLoading && <p>{t(strings.categories.load)}</p>}
        {list.error && <p className="text-sm text-app-danger">{t(strings.categories.loadError)}</p>}
        {list.data && <ul aria-label={t(strings.categories.activeList)} className="space-y-2">{list.data.categories.map((category) => <CategoryRow key={category.id} category={category} />)}</ul>}
      </CardContent>
    </Card>
  );
}
