import { Fragment } from 'react';
import { Listbox, Transition } from '@headlessui/react';
import { ChevronUpDownIcon, CheckIcon } from '@heroicons/react/24/outline';
import clsx from 'clsx';
import type { ListSummary } from '../types';

interface ListPickerProps {
  lists: ListSummary[];
  currentListId?: string;
  onSelect: (id: string) => void;
  disabled?: boolean;
}

export const ListPicker = ({ lists, currentListId, onSelect, disabled }: ListPickerProps) => {
  const current = lists.find((list) => list.id === currentListId);

  return (
    <Listbox value={current?.id ?? null} onChange={onSelect} disabled={disabled}>
      <div className={clsx('list-picker', disabled && 'list-picker--disabled')}>
        <Listbox.Button className="btn btn-secondary list-picker__button" type="button">
          <span className="list-picker__label">Active List</span>
          <span className="list-picker__value">{current?.name ?? 'Select list'}</span>
          <ChevronUpDownIcon />
        </Listbox.Button>
        <Transition
          as={Fragment}
          leave="transition ease-in duration-100"
          leaveFrom="transform opacity-100"
          leaveTo="transform opacity-0"
        >
          <Listbox.Options className="list-picker__options panel">
            {lists.length === 0 ? (
              <div className="list-picker__empty">No lists yet. Create one to get started.</div>
            ) : (
              lists.map((list) => (
                <Listbox.Option
                  key={list.id}
                  className={({ active }) =>
                    clsx('list-picker__option', active && 'list-picker__option--active')
                  }
                  value={list.id}
                >
                  {({ selected }) => (
                    <div className="list-picker__option-content">
                      <div>
                        <div className="list-picker__option-name">{list.name}</div>
                        <div className="list-picker__option-meta">{list.item_count} items</div>
                      </div>
                      {selected && <CheckIcon />}
                    </div>
                  )}
                </Listbox.Option>
              ))
            )}
          </Listbox.Options>
        </Transition>
      </div>
    </Listbox>
  );
};
