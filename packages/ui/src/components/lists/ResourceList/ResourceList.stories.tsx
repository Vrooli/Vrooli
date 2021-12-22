import { Meta, Story } from "@storybook/react";
import { ResourceList as Component } from '../';
import { ResourceListProps as Props } from '../types';

// Define story metadata
export default {
    title: 'lists/ResourceList',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});