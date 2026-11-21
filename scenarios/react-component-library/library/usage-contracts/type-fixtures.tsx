import type { ReactNode } from "react";
import {
  AppShell,
  type AppShellNavItem,
} from "../components/AppShell/versions/2.1.5/AppShell";
import {
  BottomNav,
  type BottomNavItem,
} from "../components/BottomNav/versions/1.5.7/BottomNav";
import { CollectionList } from "../components/CollectionList/versions/1.3.6/CollectionList";
import type {
  ContractCollectionItems,
  ContractItems,
} from "./generated-contract-limits";

const icon: ReactNode = null;
const navItems = [
  { id: "home", label: "Home", href: "/", icon },
  { id: "search", label: "Search", href: "/search", icon },
  { id: "saved", label: "Saved", href: "/saved", icon },
  { id: "profile", label: "Profile", href: "/profile", icon },
] satisfies readonly AppShellNavItem[];

// Valid source literal and dynamic arrays remain usable.
const validShell = (
  <AppShell brand="Demo" items={navItems}>
    <main />
  </AppShell>
);
const dynamicBottomItems: BottomNavItem[] = navItems;
const validDynamicBottomNav = (
  <BottomNav label="Primary" items={dynamicBottomItems} />
);
const validCollection = (
  <CollectionList items={[{ id: "a", title: "Alpha" }]} label="Items" />
);
const unknownItems: unknown[] = [];
const incompleteCollectionItems: { title: string }[] = [
  { title: "No stable id" },
];

const sixBottomItems = [
  { id: "1", label: "One", icon },
  { id: "2", label: "Two", icon },
  { id: "3", label: "Three", icon },
  { id: "4", label: "Four", icon },
  { id: "5", label: "Five", icon },
  { id: "6", label: "Six", icon },
] as const;
// @ts-expect-error Six source-literal items exceed BottomNav@1.5.7's maximum of five.
const tooManyBottomItems = <BottomNav label="Primary" items={sixBottomItems} />;
const missingIdItems = [{ label: "No id", icon }] as const;
const malformedBottomItem = (
  // @ts-expect-error BottomNav's required stable id field cannot be omitted.
  <BottomNav label="Primary" items={missingIdItems} />
);
// @ts-expect-error Unknown dynamic data does not satisfy BottomNav's stable item shape.
const unknownBottomItems = <BottomNav label="Primary" items={unknownItems} />;
const malformedCollection = (
  // @ts-expect-error CollectionList's contract requires its stable id and title fields.
  <CollectionList items={incompleteCollectionItems} label="Items" />
);
const multipleShellChildren = [<main key="a" />, <aside key="b" />] as const;
const tooManyShellChildren = (
  // @ts-expect-error AppShell's one direct composition slot excludes a child array.
  <AppShell brand="Demo" items={navItems} children={multipleShellChildren} />
);
type Repeat<
  Item,
  Count extends number,
  Items extends Item[] = [],
> = Items["length"] extends Count
  ? Items
  : Repeat<Item, Count, [...Items, Item]>;
type ThirteenOptions = Repeat<{ value: string; label: string }, 13>;
export type CollectionLimitRejectsThirteen = ContractCollectionItems<
  "react-component-library:AsyncOptionsField@1.0.10",
  "initialOptions",
  ThirteenOptions
>;
type MustBeNever<Value extends never> = Value;
export type CollectionMaximumIsContractDerived =
  MustBeNever<CollectionLimitRejectsThirteen>;
type ThirtyOneNavItems = Repeat<AppShellNavItem, 31>;
export type AppShellMaximumIsContractDerived = MustBeNever<
  ContractItems<"react-component-library:AppShell@2.1.5", ThirtyOneNavItems>
>;

void [
  validShell,
  validDynamicBottomNav,
  validCollection,
  tooManyBottomItems,
  malformedBottomItem,
  unknownBottomItems,
  malformedCollection,
  tooManyShellChildren,
];
