import { Meta, Story } from "@storybook/react";
import { CombineNode as Component } from '..';
import { CombineNodeProps as Props } from '../types';

// Define story metadata
export default {
    title: 'nodes/CombineNode',
    component: Component,
} as Meta;

// Define template for enabling control over props
const Template: Story<Props> = (args) => <Component {...args} />;

// Export story
export const Default = Template.bind({});