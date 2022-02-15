import { Meta, Story } from "@storybook/react";
import { SearchBar as Component } from '..';
import { SearchBarProps as Props } from '../types';

// Define story metadata
export default {
    title: 'inputs/SearchBar',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});