import type { ComponentPropsWithRef, ElementType } from "react";
import type { ContractItems, UsageContractId } from "./generated-contract-limits";

type PropsOf<Component extends ElementType> = ComponentPropsWithRef<Component>;
type CollectionItem<Component extends ElementType, Props, Property extends PropertyKey> = Property extends keyof Props
  ? NonNullable<Props[Property]> extends readonly (infer Item)[]
    ? Extract<Item, object>
    : never
  : never;
type ApplyCollectionContract<
  Props,
  Property extends PropertyKey,
  Items extends readonly unknown[],
  Contract extends UsageContractId,
> = Omit<Props, Property> & {
  [Key in Property]: Items & ContractItems<Contract, Items>;
};

/** Adds a version's JSON collection contract to its public component props. */
export type ContractedComponent<
  Component extends ElementType,
  Contract extends UsageContractId,
  Property extends PropertyKey,
> = Component extends (...args: never[]) => infer Result
  ? <const Items extends readonly CollectionItem<Component, PropsOf<Component>, Property>[]>(
      props: ApplyCollectionContract<PropsOf<Component>, Property, Items, Contract>,
    ) => Result
  : Component;
