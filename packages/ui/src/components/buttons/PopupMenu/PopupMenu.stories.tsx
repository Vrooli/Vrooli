import { Meta, Story } from "@storybook/react";
import { PopupMenu as Component } from '../';
import { PopupMenuProps as Props } from '../types';

// Define story metadata
export default {
    title: 'buttons/PopupMenu',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});