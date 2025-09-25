import { Dialog, Transition } from '@headlessui/react';
import { Fragment, type ReactNode } from 'react';

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  subtitle?: string;
  size?: 'md' | 'lg';
  footer?: ReactNode;
  children: ReactNode;
}

export const Modal = ({ open, onClose, title, subtitle, size = 'md', footer, children }: ModalProps) => {
  return (
    <Transition.Root show={open} as={Fragment}>
      <Dialog as="div" className="modal" onClose={onClose}>
        <div className="modal__backdrop" aria-hidden="true" />
        <div className="modal__container">
          <Transition.Child
            as={Fragment}
            enter="ease-out duration-200"
            enterFrom="opacity-0 translate-y-4"
            enterTo="opacity-100 translate-y-0"
            leave="ease-in duration-150"
            leaveFrom="opacity-100 translate-y-0"
            leaveTo="opacity-0 translate-y-4"
          >
            <Dialog.Panel className={`modal__panel modal__panel--${size}`}>
              <div className="modal__header">
                <div>
                  <Dialog.Title as="h2">{title}</Dialog.Title>
                  {subtitle && <p className="modal__subtitle">{subtitle}</p>}
                </div>
                <button type="button" className="modal__close" onClick={onClose}>
                  &times;
                </button>
              </div>
              <div className="modal__body">{children}</div>
              {footer && <div className="modal__footer">{footer}</div>}
            </Dialog.Panel>
          </Transition.Child>
        </div>
      </Dialog>
    </Transition.Root>
  );
};
