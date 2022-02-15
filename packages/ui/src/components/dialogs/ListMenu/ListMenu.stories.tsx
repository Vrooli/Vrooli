import { Meta, Story } from "@storybook/react";
import { ListMenu as Component } from '..';
import { ListMenuProps as Props } from '../types';

// Define story metadata
export default {
    title: 'dialogs/ListMenu',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props<string>> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});